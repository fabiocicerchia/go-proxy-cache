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
	"os"
	"strings"
	"time"
)

// ControllerName - The value an IngressClass or GatewayClass must carry in its
// `spec.controller` / `spec.controllerName` for this controller to claim it.
const ControllerName = "fabiocicerchia.it/go-proxy-cache"

// IngressClassName - Default name of the IngressClass this controller serves.
const IngressClassName = "go-proxy-cache"

// AnnotationPrefix - Prefix for every go-proxy-cache specific annotation.
const AnnotationPrefix = "go-proxy-cache.fabiocicerchia.it/"

// LegacyIngressClassAnnotation - The pre-IngressClass way of selecting a
// controller. Still widely used, so still honoured.
const LegacyIngressClassAnnotation = "kubernetes.io/ingress.class"

// DefaultIngressClassAnnotation - Marks an IngressClass as the cluster default,
// which claims Ingresses that name no class at all.
const DefaultIngressClassAnnotation = "ingressclass.kubernetes.io/is-default-class"

// DefaultResyncPeriod - How often the informers do a full relist.
const DefaultResyncPeriod = 10 * time.Minute

// DefaultElectionID - Name of the Lease coordinating status writes.
const DefaultElectionID = "go-proxy-cache-ingress-controller"

// Options - How the controller is configured.
type Options struct {
	// KubeConfig - Path to a kubeconfig file. Empty means in-cluster.
	KubeConfig string

	// ControllerName - Identity matched against IngressClass.spec.controller
	// and GatewayClass.spec.controllerName.
	ControllerName string

	// IngressClass - Name of the IngressClass to serve.
	IngressClass string

	// WatchNamespace - Restrict the controller to one namespace. Empty watches
	// the whole cluster.
	WatchNamespace string

	// PublishService - "namespace/name" of the Service whose load balancer
	// address is written into the status of the objects served.
	PublishService string

	// PublishStatusAddress - Explicit addresses to publish, bypassing
	// PublishService. Takes precedence when set.
	PublishStatusAddress []string

	// ElectionID - Name of the Lease used to elect the replica that writes
	// status. The data plane runs on every replica regardless.
	ElectionID string

	// Namespace - Namespace the controller pod runs in, where the Lease lives.
	Namespace string

	// EnableGatewayAPI - Also watch GatewayClass/Gateway/HTTPRoute.
	EnableGatewayAPI bool

	// DisableStatusUpdates - Never write status back, and never run for
	// election. Useful when RBAC is deliberately read-only.
	DisableStatusUpdates bool

	// ResyncPeriod - Informer relist interval.
	ResyncPeriod time.Duration
}

// NewOptions - Options with defaults applied, reading the environment for the
// values a Deployment normally injects.
func NewOptions() Options {
	return Options{
		ControllerName: envOr("INGRESS_CONTROLLER_NAME", ControllerName),
		IngressClass:   envOr("INGRESS_CLASS", IngressClassName),
		WatchNamespace: os.Getenv("WATCH_NAMESPACE"),
		PublishService: os.Getenv("PUBLISH_SERVICE"),
		ElectionID:     envOr("ELECTION_ID", DefaultElectionID),
		Namespace:      envOr("POD_NAMESPACE", "default"),
		ResyncPeriod:   DefaultResyncPeriod,
	}
}

// Normalise - Fills in any option left empty with its default.
func (o *Options) Normalise() {
	if o.ControllerName == "" {
		o.ControllerName = ControllerName
	}

	if o.IngressClass == "" {
		o.IngressClass = IngressClassName
	}

	if o.ElectionID == "" {
		o.ElectionID = DefaultElectionID
	}

	if o.Namespace == "" {
		o.Namespace = envOr("POD_NAMESPACE", "default")
	}

	if o.ResyncPeriod == 0 {
		o.ResyncPeriod = DefaultResyncPeriod
	}

	cleaned := make([]string, 0, len(o.PublishStatusAddress))

	for _, addr := range o.PublishStatusAddress {
		if addr = strings.TrimSpace(addr); addr != "" {
			cleaned = append(cleaned, addr)
		}
	}

	o.PublishStatusAddress = cleaned
}

func envOr(name string, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}

	return fallback
}
