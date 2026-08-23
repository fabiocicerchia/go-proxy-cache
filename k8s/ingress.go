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

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

// claimsIngress - Whether this controller is responsible for an Ingress.
//
// Three ways an Ingress selects a controller, in precedence order: the legacy
// annotation, an explicit spec.ingressClassName, or -- when it names none --
// the IngressClass marked as the cluster default.
func claimsIngress(ing *networkingv1.Ingress, classes []*networkingv1.IngressClass, opts Options) bool {
	if v, ok := ing.Annotations[LegacyIngressClassAnnotation]; ok {
		return v == opts.IngressClass
	}

	if ing.Spec.IngressClassName != nil {
		return classMatches(*ing.Spec.IngressClassName, classes, opts)
	}

	for _, class := range classes {
		if class.Annotations[DefaultIngressClassAnnotation] == "true" && class.Spec.Controller == opts.ControllerName {
			return true
		}
	}

	return false
}

func classMatches(name string, classes []*networkingv1.IngressClass, opts Options) bool {
	for _, class := range classes {
		if class.Name == name {
			return class.Spec.Controller == opts.ControllerName
		}
	}

	// The IngressClass object does not exist (yet). Fall back to the
	// configured name so a race on object creation order does not silently
	// drop the Ingress.
	return name == opts.IngressClass
}

// translateIngress - Turns one Ingress into routes.
//
// A rule that cannot be resolved (missing Service, no ready endpoints, unknown
// port) is skipped with a warning and reported through the returned problems,
// rather than failing the whole object: an Ingress with three paths where one
// backend is being redeployed must keep serving the other two.
func (c *Controller) translateIngress(ing *networkingv1.Ingress, base config.Configuration) ([]*router.Route, []string) {
	source := fmt.Sprintf("Ingress %s/%s", ing.Namespace, ing.Name)
	settings := parseAnnotations(base, ing.Annotations, source)

	routes := make([]*router.Route, 0)
	problems := make([]string, 0)

	for ruleIdx := range ing.Spec.Rules {
		rule := &ing.Spec.Rules[ruleIdx]

		if rule.HTTP == nil {
			continue
		}

		for pathIdx := range rule.HTTP.Paths {
			path := &rule.HTTP.Paths[pathIdx]

			route, err := c.ingressPathRoute(ing, rule, path, settings, ruleIdx, pathIdx)
			if err != nil {
				logger.GetGlobal().Warnf("%s: skipping path %q: %s", source, path.Path, err)
				problems = append(problems, fmt.Sprintf("path %q: %s", path.Path, err))

				continue
			}

			routes = append(routes, route)
		}
	}

	// A default backend catches everything the rules did not, so it is
	// registered as a hostless catch-all with the lowest possible precedence.
	if ing.Spec.DefaultBackend != nil {
		route, err := c.defaultBackendRoute(ing, settings)
		if err != nil {
			logger.GetGlobal().Warnf("%s: skipping default backend: %s", source, err)
			problems = append(problems, fmt.Sprintf("default backend: %s", err))
		} else {
			routes = append(routes, route)
		}
	}

	return routes, problems
}

func (c *Controller) ingressPathRoute(
	ing *networkingv1.Ingress,
	rule *networkingv1.IngressRule,
	path *networkingv1.HTTPIngressPath,
	settings routeSettings,
	ruleIdx int,
	pathIdx int,
) (*router.Route, error) {
	backend, err := c.resolveIngressBackend(ing.Namespace, &path.Backend, settings)
	if err != nil {
		return nil, err
	}

	route := c.newRoute(ing, settings, fmt.Sprintf("ingress/%s/%s/%d/%d", ing.Namespace, ing.Name, ruleIdx, pathIdx))
	route.Host = rule.Host
	route.Path = path.Path
	route.PathType = ingressPathType(path.PathType, settings)
	route.Backends = []router.Backend{backend}

	if settings.RewriteTarget != "" {
		route.Filters = append(route.Filters, rewriteFilter(settings.RewriteTarget, route.PathType))
	}

	return route, nil
}

func (c *Controller) defaultBackendRoute(ing *networkingv1.Ingress, settings routeSettings) (*router.Route, error) {
	backend, err := c.resolveIngressBackend(ing.Namespace, ing.Spec.DefaultBackend, settings)
	if err != nil {
		return nil, err
	}

	route := c.newRoute(ing, settings, fmt.Sprintf("ingress/%s/%s/default", ing.Namespace, ing.Name))
	route.Path = "/"
	route.PathType = router.PathPrefix
	route.Backends = []router.Backend{backend}

	return route, nil
}

func (c *Controller) newRoute(ing *networkingv1.Ingress, settings routeSettings, id string) *router.Route {
	return &router.Route{
		ID:                id,
		Source:            fmt.Sprintf("Ingress %s/%s", ing.Namespace, ing.Name),
		Config:            settings.Config,
		PreserveHost:      settings.PreserveHost,
		UpstreamHost:      settings.UpstreamHost,
		CreationTimestamp: ing.CreationTimestamp.Time,
	}
}

func (c *Controller) resolveIngressBackend(
	namespace string,
	backend *networkingv1.IngressBackend,
	settings routeSettings,
) (router.Backend, error) {
	if backend.Service == nil {
		return router.Backend{}, fmt.Errorf("only Service backends are supported")
	}

	port := intOrName{Number: backend.Service.Port.Number, Name: backend.Service.Port.Name}

	endpoints, err := c.backends.resolve(namespace, backend.Service.Name, port)
	if err != nil {
		return router.Backend{}, err
	}

	scheme := settings.BackendScheme
	if scheme == "" {
		scheme = "http"
	}

	return router.Backend{
		Name:      fmt.Sprintf("%s/%s:%s", namespace, backend.Service.Name, port),
		Endpoints: endpoints,
		Scheme:    scheme,
		Weight:    1,
	}, nil
}

// ingressPathType - Maps an Ingress path type onto a router path type.
//
// ImplementationSpecific is left to the controller: it is treated as a prefix,
// unless the object opts into regular expressions with the use-regex
// annotation, which is the convention users already know from ingress-nginx.
func ingressPathType(pathType *networkingv1.PathType, settings routeSettings) router.PathType {
	if pathType == nil {
		return router.PathPrefix
	}

	switch *pathType {
	case networkingv1.PathTypeExact:
		return router.PathExact
	case networkingv1.PathTypePrefix:
		return router.PathPrefix
	default:
		if settings.UseRegex {
			return router.PathRegex
		}

		return router.PathPrefix
	}
}

// rewriteFilter - Builds the URL rewrite for a rewrite-target annotation.
//
// For a prefix route the matched prefix is replaced, which is what people
// expect from `rewrite-target: /`. For an exact or regex route there is no
// meaningful prefix to strip, so the whole path is replaced.
func rewriteFilter(target string, pathType router.PathType) router.Filter {
	value := target

	if pathType == router.PathPrefix {
		return router.Filter{
			Type:    router.FilterURLRewrite,
			Rewrite: &router.Rewrite{ReplacePrefixMatch: &value},
		}
	}

	return router.Filter{
		Type:    router.FilterURLRewrite,
		Rewrite: &router.Rewrite{ReplaceFullPath: &value},
	}
}
