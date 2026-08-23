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
	"fmt"
	"reflect"
	"sort"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	networkinglisters "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gatewayinformers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
)

// syncKey - The workqueue only ever holds this one key.
//
// Every watched object feeds into a single full retranslation rather than a
// per-object reconcile: routes are inherently cross-object (an EndpointSlice
// change affects every Ingress pointing at that Service, a Gateway listener
// affects every HTTPRoute bound to it), and the whole cluster's ingress
// configuration is small enough that rebuilding it is cheaper than tracking
// the dependency graph. Coalescing on one key also means a burst of events
// collapses into a single rebuild.
const syncKey = "sync"

// resyncDebounce - How long to wait after an event before rebuilding, so a
// rollout that churns dozens of EndpointSlices produces one rebuild.
const resyncDebounce = 200 * time.Millisecond

// BalancerRegistry - What the controller needs from the server to keep load
// balancers in step with the routes. Narrow interface so the k8s package does
// not import the server package (which imports the handler, which imports the
// router: a cycle).
type BalancerRegistry interface {
	Reconcile(wanted map[string]config.Upstream)
}

// Controller - Watches the cluster and keeps the routing table, the
// certificate store and the load balancers in sync with it.
type Controller struct {
	opts Options

	core    kubernetes.Interface
	gateway gatewayclientset.Interface

	factory        informers.SharedInformerFactory
	gatewayFactory gatewayinformers.SharedInformerFactory

	ingresses      networkinglisters.IngressLister
	ingressClasses networkinglisters.IngressClassLister
	secrets        corelisters.SecretLister
	services       corelisters.ServiceLister

	backends *backendResolver
	certs    *srvtls.Store
	registry BalancerRegistry

	gatewayState *gatewayState

	queue workqueue.TypedRateLimitingInterface[string]

	// isLeader - Whether this replica writes status. The data plane runs
	// regardless.
	leader *leaderState

	// lastStatus - What was last written per object, so an unchanged status is
	// not re-sent on every resync (which would hot-loop the API server).
	lastStatus map[string]string
}

// New - Builds a controller. Does not talk to the API server yet.
func New(opts Options, certs *srvtls.Store, registry BalancerRegistry) (*Controller, error) {
	opts.Normalise()

	core, gateway, err := newClients(opts.KubeConfig)
	if err != nil {
		return nil, err
	}

	return newWithClients(opts, core, gateway, certs, registry), nil
}

func newWithClients(
	opts Options,
	core kubernetes.Interface,
	gateway gatewayclientset.Interface,
	certs *srvtls.Store,
	registry BalancerRegistry,
) *Controller {
	opts.Normalise()

	factoryOpts := []informers.SharedInformerOption{}
	if opts.WatchNamespace != "" {
		factoryOpts = append(factoryOpts, informers.WithNamespace(opts.WatchNamespace))
	}

	factory := informers.NewSharedInformerFactoryWithOptions(core, opts.ResyncPeriod, factoryOpts...)

	c := &Controller{
		opts:       opts,
		core:       core,
		gateway:    gateway,
		factory:    factory,
		certs:      certs,
		registry:   registry,
		leader:     newLeaderState(),
		lastStatus: make(map[string]string),
		queue: workqueue.NewTypedRateLimitingQueue[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
		),
	}

	c.ingresses = factory.Networking().V1().Ingresses().Lister()
	c.ingressClasses = factory.Networking().V1().IngressClasses().Lister()
	c.secrets = factory.Core().V1().Secrets().Lister()
	c.services = factory.Core().V1().Services().Lister()

	c.backends = &backendResolver{
		services:  c.services,
		endpoints: factory.Discovery().V1().EndpointSlices().Lister(),
	}

	c.watch(
		factory.Networking().V1().Ingresses().Informer(),
		factory.Networking().V1().IngressClasses().Informer(),
		factory.Core().V1().Secrets().Informer(),
		factory.Core().V1().Services().Informer(),
		factory.Discovery().V1().EndpointSlices().Informer(),
	)

	if opts.EnableGatewayAPI {
		c.setupGatewayInformers(gateway, opts)
	}

	return c
}

// watch - Enqueues a rebuild whenever any watched object changes.
func (c *Controller) watch(informers ...cache.SharedIndexInformer) {
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(interface{}) { c.enqueue() },
		UpdateFunc: func(interface{}, interface{}) { c.enqueue() },
		DeleteFunc: func(interface{}) { c.enqueue() },
	}

	for _, informer := range informers {
		// The handler is registered for its side effect; a registration only
		// fails once the informer has stopped, which cannot happen here.
		_, _ = informer.AddEventHandler(handler)
	}
}

func (c *Controller) enqueue() {
	c.queue.AddAfter(syncKey, resyncDebounce)
}

// Run - Starts the informers and serves until the context is cancelled.
func (c *Controller) Run(ctx context.Context) error {
	defer c.queue.ShutDown()

	log := logger.GetGlobal()
	log.Infof("Starting Kubernetes ingress controller (class %q, controller %q)", c.opts.IngressClass, c.opts.ControllerName)

	c.factory.Start(ctx.Done())

	if err := waitForCaches(c.factory.WaitForCacheSync(ctx.Done())); err != nil {
		return err
	}

	if c.gatewayFactory != nil {
		c.gatewayFactory.Start(ctx.Done())

		if err := waitForCaches(c.gatewayFactory.WaitForCacheSync(ctx.Done())); err != nil {
			return err
		}
	}

	log.Info("Kubernetes informer caches synced")

	if !c.opts.DisableStatusUpdates {
		go c.runLeaderElection(ctx)
	}

	// Build once up front so the proxy is serving routes before the first
	// event arrives.
	c.sync(ctx)

	go c.runWorker(ctx)

	<-ctx.Done()

	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for {
		key, shutdown := c.queue.Get()
		if shutdown {
			return
		}

		c.sync(ctx)
		c.queue.Forget(key)
		c.queue.Done(key)
	}
}

// sync - Rebuilds the whole routing table, certificate store and balancer set
// from the current informer caches.
func (c *Controller) sync(ctx context.Context) {
	log := logger.GetGlobal()

	base := config.Current().Global

	routes := make([]*router.Route, 0)
	certs := make(map[string]*tls.Certificate)

	ingresses := c.claimedIngresses()

	for _, ing := range ingresses {
		ingRoutes, problems := c.translateIngress(ing, base)
		routes = append(routes, ingRoutes...)

		c.loadIngressCertificates(ing, certs)
		c.recordIngressStatus(ctx, ing, problems)
	}

	if c.opts.EnableGatewayAPI {
		routes = append(routes, c.syncGatewayAPI(ctx, base, certs)...)
	}

	c.certs.Replace(certs)
	c.registry.Reconcile(wantedBalancers(routes))

	router.Publish(router.Build(routes))
	config.Publish(config.NewSnapshot(base, routedDomains(routes)))

	log.Debugf("Kubernetes sync: %d ingresses claimed, %d routes, %d certificates", len(ingresses), len(routes), len(certs))
}

// claimedIngresses - The Ingresses this controller is responsible for, in a
// stable order so equal-precedence routes do not shuffle between syncs.
func (c *Controller) claimedIngresses() []*networkingv1.Ingress {
	classes, err := c.ingressClasses.List(labels.Everything())
	if err != nil {
		logger.GetGlobal().Errorf("Cannot list IngressClasses: %s", err)
		classes = nil
	}

	all, err := c.ingresses.List(labels.Everything())
	if err != nil {
		logger.GetGlobal().Errorf("Cannot list Ingresses: %s", err)

		return nil
	}

	claimed := make([]*networkingv1.Ingress, 0, len(all))

	for _, ing := range all {
		if claimsIngress(ing, classes, c.opts) {
			claimed = append(claimed, ing)
		}
	}

	sort.Slice(claimed, func(i, j int) bool {
		if claimed[i].Namespace != claimed[j].Namespace {
			return claimed[i].Namespace < claimed[j].Namespace
		}

		return claimed[i].Name < claimed[j].Name
	})

	return claimed
}

// waitForCaches - Turns the informer factory's per-type sync map into an error.
func waitForCaches(synced map[reflect.Type]bool) error {
	for informerType, ok := range synced {
		if !ok {
			return fmt.Errorf("timed out waiting for %s informer cache to sync", informerType)
		}
	}

	return nil
}

// loadIngressCertificates - Reads the Secrets an Ingress references and indexes
// them by every host they serve.
func (c *Controller) loadIngressCertificates(ing *networkingv1.Ingress, into map[string]*tls.Certificate) {
	for i := range ing.Spec.TLS {
		entry := &ing.Spec.TLS[i]

		if entry.SecretName == "" {
			continue
		}

		secret, err := c.secrets.Secrets(ing.Namespace).Get(entry.SecretName)
		if err != nil {
			logger.GetGlobal().Warnf("Ingress %s/%s: cannot read TLS secret %q: %s", ing.Namespace, ing.Name, entry.SecretName, err)
			continue
		}

		cert, hosts, err := certificateFromSecret(secret)
		if err != nil {
			logger.GetGlobal().Warnf("Ingress %s/%s: invalid TLS secret %q: %s", ing.Namespace, ing.Name, entry.SecretName, err)
			continue
		}

		// The hosts listed on the Ingress win, falling back to the names in
		// the certificate itself when the Ingress lists none.
		names := entry.Hosts
		if len(names) == 0 {
			names = hosts
		}

		for _, host := range names {
			into[host] = cert
		}
	}
}

// wantedBalancers - The load balancer set the routes require.
func wantedBalancers(routes []*router.Route) map[string]config.Upstream {
	wanted := make(map[string]config.Upstream, len(routes))

	for _, route := range routes {
		for i := range route.Backends {
			wanted[route.BalancerID(i)] = route.Upstream(i)
		}
	}

	return wanted
}

// routedDomains - A domain entry per routed host.
//
// The healthcheck endpoint and the JWT middleware still resolve settings
// through config.DomainConf, so the routed hosts have to be visible there too.
func routedDomains(routes []*router.Route) config.Domains {
	domains := make(config.Domains)

	for _, route := range routes {
		if route.Host == "" {
			continue
		}

		if _, ok := domains[route.Host]; ok {
			continue
		}

		domainConfig := route.Config
		domainConfig.Domains = nil
		domainConfig.Server.Upstream.Host = route.Host
		domains[route.Host] = domainConfig
	}

	return domains
}
