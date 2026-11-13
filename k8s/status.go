package k8s

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"net"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
)

// publishAddresses - The addresses to advertise in the status of the objects
// this controller serves.
//
// Explicit addresses win, then the load balancer address of the Service named
// by --publish-service. Nothing is published when neither is available: an
// empty ADDRESS column is honest, whereas a guessed one breaks external-dns
// and cert-manager.
func (c *Controller) publishAddresses() []string {
	if len(c.opts.PublishStatusAddress) > 0 {
		return c.opts.PublishStatusAddress
	}

	if c.opts.PublishService == "" {
		return nil
	}

	namespace, name, found := strings.Cut(c.opts.PublishService, "/")
	if !found {
		logger.GetGlobal().Errorf("Invalid publish service %q, expected namespace/name", c.opts.PublishService)
		return nil
	}

	svc, err := c.services.Services(namespace).Get(name)
	if err != nil {
		logger.GetGlobal().Warnf("Cannot read publish service %s: %s", c.opts.PublishService, err)
		return nil
	}

	addresses := make([]string, 0)

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			addresses = append(addresses, ingress.IP)
		}

		if ingress.Hostname != "" {
			addresses = append(addresses, ingress.Hostname)
		}
	}

	// A ClusterIP Service has no load balancer status, but its cluster IP is
	// still the address traffic reaches the controller on.
	if len(addresses) == 0 && svc.Spec.ClusterIP != "" && svc.Spec.ClusterIP != corev1.ClusterIPNone {
		addresses = append(addresses, svc.Spec.ClusterIP)
	}

	sort.Strings(addresses)

	return addresses
}

// recordIngressStatus - Writes the load balancer addresses onto an Ingress.
//
// Skipped entirely unless this replica holds the lease, and skipped again when
// the status already says what it would say: re-sending an identical update on
// every resync would hot-loop the API server and spam the audit log.
func (c *Controller) recordIngressStatus(ctx context.Context, ing *networkingv1.Ingress, problems []string) {
	if len(problems) > 0 {
		logger.GetGlobal().Warnf("Ingress %s/%s has %d unresolved backend(s)", ing.Namespace, ing.Name, len(problems))
	}

	if c.opts.DisableStatusUpdates || !c.leader.IsLeader() {
		return
	}

	addresses := c.publishAddresses()
	if len(addresses) == 0 {
		return
	}

	key := "ingress/" + ing.Namespace + "/" + ing.Name
	want := strings.Join(addresses, ",")

	if c.lastStatus[key] == want && sameIngressStatus(ing.Status.LoadBalancer.Ingress, addresses) {
		return
	}

	if err := c.updateIngressStatus(ctx, ing, addresses); err != nil {
		logger.GetGlobal().Errorf("Cannot update status of Ingress %s/%s: %s", ing.Namespace, ing.Name, err)
		return
	}

	c.lastStatus[key] = want
}

func (c *Controller) updateIngressStatus(ctx context.Context, ing *networkingv1.Ingress, addresses []string) error {
	client := c.core.NetworkingV1().Ingresses(ing.Namespace)

	// Another replica (or a user) may write the object between the read and
	// the update; retrying on conflict is the documented way to handle it.
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, ing.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if sameIngressStatus(current.Status.LoadBalancer.Ingress, addresses) {
			return nil
		}

		current.Status.LoadBalancer.Ingress = ingressStatusEntries(addresses)

		_, err = client.UpdateStatus(ctx, current, metav1.UpdateOptions{})

		return err
	})
}

func ingressStatusEntries(addresses []string) []networkingv1.IngressLoadBalancerIngress {
	entries := make([]networkingv1.IngressLoadBalancerIngress, 0, len(addresses))

	for _, address := range addresses {
		if net.ParseIP(address) != nil {
			entries = append(entries, networkingv1.IngressLoadBalancerIngress{IP: address})
			continue
		}

		entries = append(entries, networkingv1.IngressLoadBalancerIngress{Hostname: address})
	}

	return entries
}

func sameIngressStatus(current []networkingv1.IngressLoadBalancerIngress, addresses []string) bool {
	if len(current) != len(addresses) {
		return false
	}

	existing := make([]string, 0, len(current))

	for _, entry := range current {
		if entry.IP != "" {
			existing = append(existing, entry.IP)
			continue
		}

		existing = append(existing, entry.Hostname)
	}

	sort.Strings(existing)

	want := append([]string{}, addresses...)
	sort.Strings(want)

	for i := range want {
		if want[i] != existing[i] {
			return false
		}
	}

	return true
}
