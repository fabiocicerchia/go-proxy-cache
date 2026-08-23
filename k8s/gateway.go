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
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gatewayinformers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
	gatewaylisters "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

// gatewayState - The Gateway API listers, kept together so the core controller
// carries a single nil-able field when Gateway API support is off.
type gatewayState struct {
	classes  gatewaylisters.GatewayClassLister
	gateways gatewaylisters.GatewayLister
	routes   gatewaylisters.HTTPRouteLister
}

func (c *Controller) setupGatewayInformers(client gatewayclientset.Interface, opts Options) {
	factoryOpts := []gatewayinformers.SharedInformerOption{}
	if opts.WatchNamespace != "" {
		factoryOpts = append(factoryOpts, gatewayinformers.WithNamespace(opts.WatchNamespace))
	}

	factory := gatewayinformers.NewSharedInformerFactoryWithOptions(client, opts.ResyncPeriod, factoryOpts...)

	c.gatewayFactory = factory
	c.gatewayState = &gatewayState{
		classes:  factory.Gateway().V1().GatewayClasses().Lister(),
		gateways: factory.Gateway().V1().Gateways().Lister(),
		routes:   factory.Gateway().V1().HTTPRoutes().Lister(),
	}

	c.watch(
		factory.Gateway().V1().GatewayClasses().Informer(),
		factory.Gateway().V1().Gateways().Informer(),
		factory.Gateway().V1().HTTPRoutes().Informer(),
	)
}

// syncGatewayAPI - Translates the claimed Gateways and the HTTPRoutes bound to
// them into routes, and collects their certificates.
func (c *Controller) syncGatewayAPI(ctx context.Context, base config.Configuration, certs map[string]*tls.Certificate) []*router.Route {
	gateways := c.claimedGateways()
	if len(gateways) == 0 {
		return nil
	}

	for _, gw := range gateways {
		c.loadGatewayCertificates(gw, certs)
	}

	httpRoutes, err := c.gatewayState.routes.List(labels.Everything())
	if err != nil {
		logger.GetGlobal().Errorf("Cannot list HTTPRoutes: %s", err)
		return nil
	}

	sort.Slice(httpRoutes, func(i, j int) bool {
		if httpRoutes[i].Namespace != httpRoutes[j].Namespace {
			return httpRoutes[i].Namespace < httpRoutes[j].Namespace
		}

		return httpRoutes[i].Name < httpRoutes[j].Name
	})

	routes := make([]*router.Route, 0)
	attached := make(map[string]int32)

	for _, hr := range httpRoutes {
		parents := make([]gatewayv1.RouteParentStatus, 0)

		for _, gw := range gateways {
			listeners := boundListeners(hr, gw)
			if len(listeners) == 0 {
				continue
			}

			attached[gatewayKey(gw)] += int32(len(listeners))

			hrRoutes := c.translateHTTPRoute(hr, gw, listeners, base)
			routes = append(routes, hrRoutes...)

			// Every rule producing at least one route means every backend
			// resolved; anything less is reported as ResolvedRefs=False so
			// `kubectl describe httproute` says which side is broken.
			resolved := len(hrRoutes) > 0 || len(hr.Spec.Rules) == 0
			parents = append(parents, routeParentStatus(c.opts.ControllerName, gw, hr, resolved))
		}

		if len(parents) > 0 {
			c.recordHTTPRouteStatus(ctx, hr, parents)
		}
	}

	c.recordGatewayStatus(ctx, gateways, attached)

	return routes
}

// claimedGateways - Gateways whose GatewayClass names this controller.
func (c *Controller) claimedGateways() []*gatewayv1.Gateway {
	classes, err := c.gatewayState.classes.List(labels.Everything())
	if err != nil {
		logger.GetGlobal().Errorf("Cannot list GatewayClasses: %s", err)
		return nil
	}

	ours := make(map[string]bool)

	for _, class := range classes {
		if string(class.Spec.ControllerName) == c.opts.ControllerName {
			ours[class.Name] = true
		}
	}

	if len(ours) == 0 {
		return nil
	}

	all, err := c.gatewayState.gateways.List(labels.Everything())
	if err != nil {
		logger.GetGlobal().Errorf("Cannot list Gateways: %s", err)
		return nil
	}

	claimed := make([]*gatewayv1.Gateway, 0, len(all))

	for _, gw := range all {
		if ours[string(gw.Spec.GatewayClassName)] {
			claimed = append(claimed, gw)
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

// boundListeners - The Gateway listeners an HTTPRoute actually attaches to.
//
// A route binds when it names the Gateway in a parentRef AND the listener
// admits it: the section name matches (when given), the protocol is HTTP-ish,
// and the listener's allowedRoutes permits the route's namespace.
func boundListeners(hr *gatewayv1.HTTPRoute, gw *gatewayv1.Gateway) []*gatewayv1.Listener {
	listeners := make([]*gatewayv1.Listener, 0)

	for i := range hr.Spec.ParentRefs {
		ref := &hr.Spec.ParentRefs[i]

		if !parentRefMatches(ref, hr.Namespace, gw) {
			continue
		}

		for j := range gw.Spec.Listeners {
			listener := &gw.Spec.Listeners[j]

			if listener.Protocol != gatewayv1.HTTPProtocolType && listener.Protocol != gatewayv1.HTTPSProtocolType {
				continue
			}

			if ref.SectionName != nil && string(*ref.SectionName) != string(listener.Name) {
				continue
			}

			if !listenerAllowsNamespace(listener, gw.Namespace, hr.Namespace) {
				continue
			}

			listeners = append(listeners, listener)
		}
	}

	return listeners
}

func parentRefMatches(ref *gatewayv1.ParentReference, routeNamespace string, gw *gatewayv1.Gateway) bool {
	if ref.Group != nil && *ref.Group != "" && string(*ref.Group) != gatewayv1.GroupName {
		return false
	}

	if ref.Kind != nil && *ref.Kind != "Gateway" {
		return false
	}

	namespace := routeNamespace
	if ref.Namespace != nil {
		namespace = string(*ref.Namespace)
	}

	return string(ref.Name) == gw.Name && namespace == gw.Namespace
}

// listenerAllowsNamespace - Whether a listener admits routes from a namespace.
//
// Selector-based policies need a namespace lister to evaluate labels; until
// that is wired the conservative reading is to refuse, so a route is never
// attached to a listener that did not clearly ask for it.
func listenerAllowsNamespace(listener *gatewayv1.Listener, gatewayNamespace string, routeNamespace string) bool {
	if listener.AllowedRoutes == nil || listener.AllowedRoutes.Namespaces == nil || listener.AllowedRoutes.Namespaces.From == nil {
		return routeNamespace == gatewayNamespace
	}

	switch *listener.AllowedRoutes.Namespaces.From {
	case gatewayv1.NamespacesFromAll:
		return true
	case gatewayv1.NamespacesFromSame:
		return routeNamespace == gatewayNamespace
	default:
		return false
	}
}

// translateHTTPRoute - Turns an HTTPRoute into routes, one per (hostname,
// match) pair.
func (c *Controller) translateHTTPRoute(
	hr *gatewayv1.HTTPRoute,
	gw *gatewayv1.Gateway,
	listeners []*gatewayv1.Listener,
	base config.Configuration,
) []*router.Route {
	source := fmt.Sprintf("HTTPRoute %s/%s", hr.Namespace, hr.Name)
	settings := parseAnnotations(base, hr.Annotations, source)

	hostnames := effectiveHostnames(hr, listeners)
	routes := make([]*router.Route, 0)

	for ruleIdx := range hr.Spec.Rules {
		rule := &hr.Spec.Rules[ruleIdx]

		backends := c.resolveHTTPRouteBackends(hr, rule, settings)
		if len(backends) == 0 {
			logger.GetGlobal().Warnf("%s: rule %d has no resolvable backend", source, ruleIdx)
			continue
		}

		filters := translateHTTPFilters(rule.Filters)
		matches := rule.Matches

		// A rule with no matches serves every path, per spec.
		if len(matches) == 0 {
			matches = []gatewayv1.HTTPRouteMatch{{}}
		}

		for matchIdx := range matches {
			for hostIdx, hostname := range hostnames {
				route := &router.Route{
					ID: fmt.Sprintf("httproute/%s/%s/%s/%d/%d/%d",
						hr.Namespace, hr.Name, gatewayKey(gw), ruleIdx, matchIdx, hostIdx),
					Source:            source,
					Host:              hostname,
					Config:            settings.Config,
					PreserveHost:      settings.PreserveHost,
					UpstreamHost:      settings.UpstreamHost,
					Backends:          backends,
					Filters:           filters,
					CreationTimestamp: hr.CreationTimestamp.Time,
				}

				applyHTTPMatch(route, &matches[matchIdx])
				routes = append(routes, route)
			}
		}
	}

	return routes
}

// effectiveHostnames - The intersection of the route's hostnames with the
// listeners', which is what the route actually serves.
func effectiveHostnames(hr *gatewayv1.HTTPRoute, listeners []*gatewayv1.Listener) []string {
	listenerHosts := make([]string, 0, len(listeners))

	for _, listener := range listeners {
		if listener.Hostname != nil && *listener.Hostname != "" {
			listenerHosts = append(listenerHosts, string(*listener.Hostname))
			continue
		}

		listenerHosts = append(listenerHosts, "")
	}

	routeHosts := make([]string, 0, len(hr.Spec.Hostnames))
	for _, h := range hr.Spec.Hostnames {
		routeHosts = append(routeHosts, string(h))
	}

	if len(routeHosts) == 0 {
		return dedupe(listenerHosts)
	}

	out := make([]string, 0)

	for _, routeHost := range routeHosts {
		for _, listenerHost := range listenerHosts {
			if host, ok := intersectHostnames(routeHost, listenerHost); ok {
				out = append(out, host)
			}
		}
	}

	return dedupe(out)
}

// intersectHostnames - The more specific of two hostnames, when one covers the
// other. "" means "any host".
func intersectHostnames(a string, b string) (string, bool) {
	switch {
	case a == "":
		return b, true
	case b == "":
		return a, true
	case a == b:
		return a, true
	case strings.HasPrefix(a, "*.") && hostCoveredByWildcard(a, b):
		return b, true
	case strings.HasPrefix(b, "*.") && hostCoveredByWildcard(b, a):
		return a, true
	default:
		return "", false
	}
}

func hostCoveredByWildcard(wildcard string, host string) bool {
	suffix := wildcard[1:]

	if !strings.HasSuffix(host, suffix) {
		return false
	}

	label := host[:len(host)-len(suffix)]

	return label != "" && !strings.Contains(label, ".")
}

func dedupe(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))

	for _, v := range values {
		if seen[v] {
			continue
		}

		seen[v] = true

		out = append(out, v)
	}

	return out
}

func (c *Controller) resolveHTTPRouteBackends(
	hr *gatewayv1.HTTPRoute,
	rule *gatewayv1.HTTPRouteRule,
	settings routeSettings,
) []router.Backend {
	backends := make([]router.Backend, 0, len(rule.BackendRefs))

	for i := range rule.BackendRefs {
		ref := &rule.BackendRefs[i]

		if ref.Kind != nil && *ref.Kind != "Service" {
			logger.GetGlobal().Warnf("HTTPRoute %s/%s: unsupported backend kind %q", hr.Namespace, hr.Name, *ref.Kind)
			continue
		}

		namespace := hr.Namespace
		if ref.Namespace != nil {
			namespace = string(*ref.Namespace)
		}

		port := intOrName{}
		if ref.Port != nil {
			port.Number = int32(*ref.Port)
		}

		endpoints, err := c.backends.resolve(namespace, string(ref.Name), port)
		if err != nil {
			logger.GetGlobal().Warnf("HTTPRoute %s/%s: cannot resolve backend %s/%s: %s", hr.Namespace, hr.Name, namespace, ref.Name, err)
			continue
		}

		scheme := settings.BackendScheme
		if scheme == "" {
			scheme = "http"
		}

		weight := int32(1)
		if ref.Weight != nil {
			weight = *ref.Weight
		}

		backends = append(backends, router.Backend{
			Name:      fmt.Sprintf("%s/%s:%s", namespace, ref.Name, port),
			Endpoints: endpoints,
			Scheme:    scheme,
			Weight:    weight,
		})
	}

	return backends
}

func applyHTTPMatch(route *router.Route, match *gatewayv1.HTTPRouteMatch) {
	route.Path = "/"
	route.PathType = router.PathPrefix

	if match.Path != nil {
		if match.Path.Value != nil {
			route.Path = *match.Path.Value
		}

		if match.Path.Type != nil {
			switch *match.Path.Type {
			case gatewayv1.PathMatchExact:
				route.PathType = router.PathExact
			case gatewayv1.PathMatchRegularExpression:
				route.PathType = router.PathRegex
			default:
				route.PathType = router.PathPrefix
			}
		}
	}

	if match.Method != nil {
		route.Methods = []string{string(*match.Method)}
	}

	for i := range match.Headers {
		route.Headers = append(route.Headers, router.Match{
			Name:  string(match.Headers[i].Name),
			Value: match.Headers[i].Value,
			Type:  headerMatchType(match.Headers[i].Type),
		})
	}

	for i := range match.QueryParams {
		route.Query = append(route.Query, router.Match{
			Name:  string(match.QueryParams[i].Name),
			Value: match.QueryParams[i].Value,
			Type:  queryMatchType(match.QueryParams[i].Type),
		})
	}
}

func headerMatchType(t *gatewayv1.HeaderMatchType) router.MatchType {
	if t != nil && *t == gatewayv1.HeaderMatchRegularExpression {
		return router.MatchRegex
	}

	return router.MatchExact
}

func queryMatchType(t *gatewayv1.QueryParamMatchType) router.MatchType {
	if t != nil && *t == gatewayv1.QueryParamMatchRegularExpression {
		return router.MatchRegex
	}

	return router.MatchExact
}

func translateHTTPFilters(filters []gatewayv1.HTTPRouteFilter) []router.Filter {
	out := make([]router.Filter, 0, len(filters))

	for i := range filters {
		f := &filters[i]

		switch f.Type {
		case gatewayv1.HTTPRouteFilterRequestHeaderModifier:
			if f.RequestHeaderModifier != nil {
				out = append(out, router.Filter{
					Type:           router.FilterRequestHeaderModifier,
					RequestHeaders: headerOp(f.RequestHeaderModifier),
				})
			}
		case gatewayv1.HTTPRouteFilterResponseHeaderModifier:
			if f.ResponseHeaderModifier != nil {
				out = append(out, router.Filter{
					Type:            router.FilterResponseHeaderModifier,
					ResponseHeaders: headerOp(f.ResponseHeaderModifier),
				})
			}
		case gatewayv1.HTTPRouteFilterRequestRedirect:
			if f.RequestRedirect != nil {
				out = append(out, router.Filter{
					Type:     router.FilterRequestRedirect,
					Redirect: redirectFrom(f.RequestRedirect),
				})
			}
		case gatewayv1.HTTPRouteFilterURLRewrite:
			if f.URLRewrite != nil {
				out = append(out, router.Filter{
					Type:    router.FilterURLRewrite,
					Rewrite: rewriteFrom(f.URLRewrite),
				})
			}
		}
	}

	return out
}

func headerOp(m *gatewayv1.HTTPHeaderFilter) *router.HeaderOp {
	op := &router.HeaderOp{
		Set:    make(map[string]string, len(m.Set)),
		Add:    make(map[string]string, len(m.Add)),
		Remove: m.Remove,
	}

	for _, h := range m.Set {
		op.Set[string(h.Name)] = h.Value
	}

	for _, h := range m.Add {
		op.Add[string(h.Name)] = h.Value
	}

	return op
}

func redirectFrom(r *gatewayv1.HTTPRequestRedirectFilter) *router.Redirect {
	out := &router.Redirect{}

	if r.Scheme != nil {
		out.Scheme = *r.Scheme
	}

	if r.Hostname != nil {
		out.Hostname = string(*r.Hostname)
	}

	if r.Port != nil {
		out.Port = int(*r.Port)
	}

	if r.StatusCode != nil {
		out.StatusCode = *r.StatusCode
	}

	if r.Path != nil {
		out.Path = pathModifier(r.Path)
	}

	return out
}

func rewriteFrom(r *gatewayv1.HTTPURLRewriteFilter) *router.Rewrite {
	out := &router.Rewrite{}

	if r.Hostname != nil {
		out.Hostname = string(*r.Hostname)
	}

	if r.Path != nil {
		modifier := pathModifier(r.Path)
		out.ReplaceFullPath = modifier.ReplaceFullPath
		out.ReplacePrefixMatch = modifier.ReplacePrefixMatch
	}

	return out
}

func pathModifier(p *gatewayv1.HTTPPathModifier) *router.Rewrite {
	out := &router.Rewrite{}

	switch p.Type {
	case gatewayv1.FullPathHTTPPathModifier:
		out.ReplaceFullPath = p.ReplaceFullPath
	case gatewayv1.PrefixMatchHTTPPathModifier:
		out.ReplacePrefixMatch = p.ReplacePrefixMatch
	}

	return out
}

func gatewayKey(gw *gatewayv1.Gateway) string {
	return gw.Namespace + "/" + gw.Name
}
