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
	"fmt"
	"net"
	"sort"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	discoverylisters "k8s.io/client-go/listers/discovery/v1"
)

// backendResolver - Turns a Service reference into the concrete addresses to
// proxy to.
//
// Endpoints come from EndpointSlices (ready pod IPs) rather than from the
// Service ClusterIP: routing through kube-proxy would flatten the load
// balancing to L4 round-robin, hide per-pod metrics, and react to readiness
// only as fast as kube-proxy reprograms its rules. Talking to pods directly is
// what every other ingress controller does, and it lets the existing
// balancing algorithms actually do their job.
type backendResolver struct {
	services  corelisters.ServiceLister
	endpoints discoverylisters.EndpointSliceLister
}

// resolve - Endpoints for a Service port, expressed as "host:port" strings.
//
// The named or numeric port is resolved through the Service, because an
// EndpointSlice carries the target (container) port, which frequently differs
// from the Service port the Ingress refers to.
func (b *backendResolver) resolve(namespace string, name string, port intOrName) ([]string, error) {
	svc, err := b.services.Services(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	svcPort, err := findServicePort(svc, port)
	if err != nil {
		return nil, err
	}

	// An ExternalName Service has no endpoints of its own: it is a CNAME, so
	// the only thing to proxy to is the name it points at.
	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		return []string{net.JoinHostPort(svc.Spec.ExternalName, strconv.Itoa(int(svcPort.Port)))}, nil
	}

	slices, err := b.endpoints.EndpointSlices(namespace).List(labels.SelectorFromSet(labels.Set{
		discoveryv1.LabelServiceName: name,
	}))
	if err != nil {
		return nil, err
	}

	addresses := endpointAddresses(slices, svcPort)

	if len(addresses) == 0 {
		// No slices yet (or every pod unready): fall back to the in-cluster
		// Service DNS name so the route still serves rather than 404ing while
		// EndpointSlices catch up.
		if svc.Spec.ClusterIP != "" && svc.Spec.ClusterIP != corev1.ClusterIPNone {
			return []string{net.JoinHostPort(
				fmt.Sprintf("%s.%s.svc", name, namespace),
				strconv.Itoa(int(svcPort.Port)),
			)}, nil
		}

		return nil, fmt.Errorf("no ready endpoints for service %s/%s", namespace, name)
	}

	return addresses, nil
}

func endpointAddresses(slices []*discoveryv1.EndpointSlice, svcPort *corev1.ServicePort) []string {
	addresses := make([]string, 0)

	for _, slice := range slices {
		// IPv4 and IPv6 slices are separate objects for the same Service;
		// FQDN slices carry names rather than addresses.
		if slice.AddressType != discoveryv1.AddressTypeIPv4 &&
			slice.AddressType != discoveryv1.AddressTypeIPv6 {
			continue
		}

		targetPort, ok := sliceTargetPort(slice, svcPort)
		if !ok {
			continue
		}

		for i := range slice.Endpoints {
			endpoint := &slice.Endpoints[i]

			// Ready is a *bool where nil means ready, per the API contract.
			if endpoint.Conditions.Ready != nil && !*endpoint.Conditions.Ready {
				continue
			}

			// Terminating pods still appear in the slice; sending them new
			// connections is what causes 502s during a rollout.
			if endpoint.Conditions.Terminating != nil && *endpoint.Conditions.Terminating {
				continue
			}

			for _, addr := range endpoint.Addresses {
				addresses = append(addresses, net.JoinHostPort(addr, strconv.Itoa(int(targetPort))))
			}
		}
	}

	// Stable ordering keeps the balancer's rotation from being reshuffled on
	// every resync, and makes "did the backend actually change?" comparable.
	sort.Strings(addresses)

	return addresses
}

func sliceTargetPort(slice *discoveryv1.EndpointSlice, svcPort *corev1.ServicePort) (int32, bool) {
	for i := range slice.Ports {
		p := &slice.Ports[i]

		if p.Port == nil {
			continue
		}

		// A single-port Service leaves the port name empty on both sides.
		name := ""
		if p.Name != nil {
			name = *p.Name
		}

		if name == svcPort.Name {
			return *p.Port, true
		}
	}

	return 0, false
}

// intOrName - A Service port referenced either by number or by name, which is
// how both Ingress and HTTPRoute express it.
type intOrName struct {
	Number int32
	Name   string
}

func findServicePort(svc *corev1.Service, want intOrName) (*corev1.ServicePort, error) {
	for i := range svc.Spec.Ports {
		p := &svc.Spec.Ports[i]

		if want.Name != "" && p.Name == want.Name {
			return p, nil
		}

		if want.Name == "" && p.Port == want.Number {
			return p, nil
		}
	}

	return nil, fmt.Errorf("service %s/%s has no port %s", svc.Namespace, svc.Name, want)
}

func (p intOrName) String() string {
	if p.Name != "" {
		return p.Name
	}

	return strconv.Itoa(int(p.Number))
}
