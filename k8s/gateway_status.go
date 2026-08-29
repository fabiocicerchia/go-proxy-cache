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
	"crypto/tls"
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
)

// loadGatewayCertificates - Reads the Secrets a Gateway's HTTPS listeners
// reference, indexing each certificate by the listener hostname (and, when the
// listener has none, by the names in the certificate itself).
func (c *Controller) loadGatewayCertificates(gw *gatewayv1.Gateway, into map[string]*tls.Certificate) {
	for i := range gw.Spec.Listeners {
		listener := &gw.Spec.Listeners[i]

		if listener.TLS == nil {
			continue
		}

		for j := range listener.TLS.CertificateRefs {
			ref := &listener.TLS.CertificateRefs[j]

			if ref.Kind != nil && *ref.Kind != "Secret" {
				continue
			}

			namespace := gw.Namespace
			if ref.Namespace != nil {
				namespace = string(*ref.Namespace)
			}

			secret, err := c.secrets.Secrets(namespace).Get(string(ref.Name))
			if err != nil {
				logger.GetGlobal().Warnf("Gateway %s: cannot read TLS secret %s/%s: %s", gatewayKey(gw), namespace, ref.Name, err)
				continue
			}

			cert, hosts, err := certificateFromSecret(secret)
			if err != nil {
				logger.GetGlobal().Warnf("Gateway %s: invalid TLS secret %s/%s: %s", gatewayKey(gw), namespace, ref.Name, err)
				continue
			}

			if listener.Hostname != nil && *listener.Hostname != "" {
				into[string(*listener.Hostname)] = cert
				continue
			}

			for _, host := range hosts {
				into[host] = cert
			}
		}
	}
}

// recordGatewayStatus - Publishes addresses and readiness conditions on the
// Gateways this controller serves.
func (c *Controller) recordGatewayStatus(ctx context.Context, gateways []*gatewayv1.Gateway, attached map[string]int32) {
	if c.opts.DisableStatusUpdates || !c.leader.IsLeader() {
		return
	}

	addresses := c.publishAddresses()

	for _, gw := range gateways {
		if err := c.updateGatewayStatus(ctx, gw, addresses, attached[gatewayKey(gw)]); err != nil {
			logger.GetGlobal().Errorf("Cannot update status of Gateway %s: %s", gatewayKey(gw), err)
		}
	}
}

func (c *Controller) updateGatewayStatus(
	ctx context.Context,
	gw *gatewayv1.Gateway,
	addresses []string,
	attachedRoutes int32,
) error {
	client := c.gateway.GatewayV1().Gateways(gw.Namespace)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, gw.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		desired := gatewayStatusFor(current, addresses, attachedRoutes)
		if sameGatewayStatus(current.Status, desired) {
			return nil
		}

		current.Status = desired

		_, err = client.UpdateStatus(ctx, current, metav1.UpdateOptions{})

		return err
	})
}

func gatewayStatusFor(gw *gatewayv1.Gateway, addresses []string, attachedRoutes int32) gatewayv1.GatewayStatus {
	status := gatewayv1.GatewayStatus{
		Addresses:  gatewayAddresses(addresses),
		Conditions: gatewayConditions(gw.Generation),
	}

	for i := range gw.Spec.Listeners {
		listener := &gw.Spec.Listeners[i]

		supported := listener.Protocol == gatewayv1.HTTPProtocolType || listener.Protocol == gatewayv1.HTTPSProtocolType

		status.Listeners = append(status.Listeners, gatewayv1.ListenerStatus{
			Name:           listener.Name,
			SupportedKinds: []gatewayv1.RouteGroupKind{{Kind: "HTTPRoute"}},
			AttachedRoutes: attachedRoutes,
			Conditions:     listenerConditions(gw.Generation, supported),
		})
	}

	return status
}

func gatewayAddresses(addresses []string) []gatewayv1.GatewayStatusAddress {
	out := make([]gatewayv1.GatewayStatusAddress, 0, len(addresses))

	for _, address := range addresses {
		addressType := gatewayv1.HostnameAddressType
		if net.ParseIP(address) != nil {
			addressType = gatewayv1.IPAddressType
		}

		out = append(out, gatewayv1.GatewayStatusAddress{
			Type:  &addressType,
			Value: address,
		})
	}

	return out
}

func gatewayConditions(generation int64) []metav1.Condition {
	return []metav1.Condition{
		condition(string(gatewayv1.GatewayConditionAccepted), metav1.ConditionTrue,
			string(gatewayv1.GatewayReasonAccepted), "Gateway is served by go-proxy-cache", generation),
		condition(string(gatewayv1.GatewayConditionProgrammed), metav1.ConditionTrue,
			string(gatewayv1.GatewayReasonProgrammed), "Listeners are configured on the data plane", generation),
	}
}

func listenerConditions(generation int64, supported bool) []metav1.Condition {
	if !supported {
		return []metav1.Condition{
			condition(string(gatewayv1.ListenerConditionAccepted), metav1.ConditionFalse,
				string(gatewayv1.ListenerReasonUnsupportedProtocol),
				"Only the HTTP and HTTPS protocols are supported", generation),
		}
	}

	return []metav1.Condition{
		condition(string(gatewayv1.ListenerConditionAccepted), metav1.ConditionTrue,
			string(gatewayv1.ListenerReasonAccepted), "Listener is accepted", generation),
		condition(string(gatewayv1.ListenerConditionProgrammed), metav1.ConditionTrue,
			string(gatewayv1.ListenerReasonProgrammed), "Listener is configured on the data plane", generation),
		condition(string(gatewayv1.ListenerConditionResolvedRefs), metav1.ConditionTrue,
			string(gatewayv1.ListenerReasonResolvedRefs), "All references resolved", generation),
	}
}

func condition(condType string, status metav1.ConditionStatus, reason string, message string, generation int64) metav1.Condition {
	return metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: generation,
		// LastTransitionTime is deliberately zero: it is filled in by the API
		// server, and stamping it here would make every status differ from the
		// last one and re-trigger an update on every resync.
	}
}

// sameGatewayStatus - Whether a computed status says the same thing as the one
// already on the object, ignoring the timestamps the API server maintains.
func sameGatewayStatus(current gatewayv1.GatewayStatus, desired gatewayv1.GatewayStatus) bool {
	if len(current.Addresses) != len(desired.Addresses) ||
		len(current.Listeners) != len(desired.Listeners) ||
		!sameConditions(current.Conditions, desired.Conditions) {
		return false
	}

	for i := range current.Addresses {
		if current.Addresses[i].Value != desired.Addresses[i].Value {
			return false
		}
	}

	for i := range current.Listeners {
		if current.Listeners[i].Name != desired.Listeners[i].Name ||
			current.Listeners[i].AttachedRoutes != desired.Listeners[i].AttachedRoutes ||
			!sameConditions(current.Listeners[i].Conditions, desired.Listeners[i].Conditions) {
			return false
		}
	}

	return true
}

func sameConditions(current []metav1.Condition, desired []metav1.Condition) bool {
	if len(current) != len(desired) {
		return false
	}

	for i := range desired {
		found := false

		for j := range current {
			if current[j].Type != desired[i].Type {
				continue
			}

			found = current[j].Status == desired[i].Status &&
				current[j].Reason == desired[i].Reason &&
				current[j].ObservedGeneration == desired[i].ObservedGeneration

			break
		}

		if !found {
			return false
		}
	}

	return true
}

// recordHTTPRouteStatus - Publishes the parent status for the Gateways an
// HTTPRoute is attached to.
func (c *Controller) recordHTTPRouteStatus(ctx context.Context, hr *gatewayv1.HTTPRoute, parents []gatewayv1.RouteParentStatus) {
	if c.opts.DisableStatusUpdates || !c.leader.IsLeader() {
		return
	}

	client := c.gateway.GatewayV1().HTTPRoutes(hr.Namespace)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, hr.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// Only this controller's parent entries are ours to rewrite: an
		// HTTPRoute can be attached to several controllers at once, and
		// clobbering their entries would break them.
		merged := make([]gatewayv1.RouteParentStatus, 0, len(current.Status.Parents))

		for _, parent := range current.Status.Parents {
			if string(parent.ControllerName) != c.opts.ControllerName {
				merged = append(merged, parent)
			}
		}

		merged = append(merged, parents...)

		if sameParents(current.Status.Parents, merged) {
			return nil
		}

		current.Status.Parents = merged

		_, err = client.UpdateStatus(ctx, current, metav1.UpdateOptions{})

		return err
	})
	if err != nil {
		logger.GetGlobal().Errorf("Cannot update status of HTTPRoute %s/%s: %s", hr.Namespace, hr.Name, err)
	}
}

func sameParents(current []gatewayv1.RouteParentStatus, desired []gatewayv1.RouteParentStatus) bool {
	if len(current) != len(desired) {
		return false
	}

	for i := range desired {
		if current[i].ControllerName != desired[i].ControllerName ||
			current[i].ParentRef.Name != desired[i].ParentRef.Name ||
			!sameConditions(current[i].Conditions, desired[i].Conditions) {
			return false
		}
	}

	return true
}

// routeParentStatus - The status entry for one Gateway an HTTPRoute attached to.
func routeParentStatus(controllerName string, gw *gatewayv1.Gateway, hr *gatewayv1.HTTPRoute, resolved bool) gatewayv1.RouteParentStatus {
	namespace := gatewayv1.Namespace(gw.Namespace)

	conditions := []metav1.Condition{
		condition(string(gatewayv1.RouteConditionAccepted), metav1.ConditionTrue,
			string(gatewayv1.RouteReasonAccepted), "Route is served by go-proxy-cache", hr.Generation),
	}

	if resolved {
		conditions = append(conditions, condition(string(gatewayv1.RouteConditionResolvedRefs), metav1.ConditionTrue,
			string(gatewayv1.RouteReasonResolvedRefs), "All references resolved", hr.Generation))
	} else {
		conditions = append(conditions, condition(string(gatewayv1.RouteConditionResolvedRefs), metav1.ConditionFalse,
			string(gatewayv1.RouteReasonBackendNotFound), "One or more backends could not be resolved", hr.Generation))
	}

	return gatewayv1.RouteParentStatus{
		ParentRef: gatewayv1.ParentReference{
			Name:      gatewayv1.ObjectName(gw.Name),
			Namespace: &namespace,
		},
		ControllerName: gatewayv1.GatewayController(controllerName),
		Conditions:     conditions,
	}
}
