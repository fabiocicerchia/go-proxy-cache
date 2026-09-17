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
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

// certificateOwner - Which object supplied the certificate for a hostname.
//
// Kept alongside the certificate map so a second object claiming the same
// hostname can be reported instead of silently replacing the first.
type certificateOwner map[string]string

// claim - Records who serves a hostname, warning when someone already did.
func (o certificateOwner) claim(host string, source string) {
	if previous, taken := o[host]; taken && previous != source {
		logger.GetGlobal().Warnf(
			"Certificate for %s is claimed by both %s and %s; %s wins, since it sorts later",
			host, previous, source, source,
		)
	}

	o[host] = source
}

// routeConflict - Two objects serving the same host, path and path type.
type routeConflict struct {
	Host   string
	Path   string
	Winner *router.Route
	Loser  *router.Route
}

func (c routeConflict) String() string {
	return fmt.Sprintf("%s%s is served by both %s and %s", c.Host, c.Path, c.Winner.Source, c.Loser.Source)
}

// findRouteConflicts - Routes that collide on host, path and path type.
//
// Two objects claiming one host and path is legal as far as the Ingress API is
// concerned -- nothing in it expresses hostname ownership -- but only one of
// them can win, and which one comes down to sort order. Silently dropping the
// other leaves an operator with a route that simply does not work and no
// indication why, so the collision is reported on both objects.
//
// The winner is the route that sorts first under the same precedence rules the
// routing table applies, so this says what the proxy will actually do.
func findRouteConflicts(routes []*router.Route) []routeConflict {
	type key struct {
		host     string
		path     string
		pathType router.PathType
	}

	grouped := make(map[key][]*router.Route)

	for _, route := range routes {
		k := key{host: route.Host, path: route.Path, pathType: route.PathType}
		grouped[k] = append(grouped[k], route)
	}

	conflicts := make([]routeConflict, 0)

	for k, group := range grouped {
		if len(group) < 2 {
			continue
		}

		// Same order the table itself resolves in, so the winner named here is
		// the one that will serve the traffic.
		ordered := make([]*router.Route, len(group))
		copy(ordered, group)
		router.SortRoutes(ordered)

		for _, loser := range ordered[1:] {
			if loser.Source == ordered[0].Source {
				continue
			}

			conflicts = append(conflicts, routeConflict{
				Host:   k.host,
				Path:   k.path,
				Winner: ordered[0],
				Loser:  loser,
			})
		}
	}

	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].Host != conflicts[j].Host {
			return conflicts[i].Host < conflicts[j].Host
		}

		if conflicts[i].Path != conflicts[j].Path {
			return conflicts[i].Path < conflicts[j].Path
		}

		return conflicts[i].Loser.Source < conflicts[j].Loser.Source
	})

	return conflicts
}

// dropCatchAllRoutes - Removes routes that match every hostname.
//
// A hostless Ingress rule, or a defaultBackend, captures every path no other
// route claims -- on every hostname in the cluster, whoever owns them. That is
// useful for a single-tenant install and dangerous for a shared one, so it can
// be turned off, the same way ingress-nginx ships --disable-catch-all.
func dropCatchAllRoutes(routes []*router.Route) []*router.Route {
	kept := make([]*router.Route, 0, len(routes))

	for _, route := range routes {
		if route.Host == "" {
			logger.GetGlobal().Warnf(
				"Ignoring catch-all route from %s: it would serve every hostname (catch-all routes are disabled)",
				route.Source,
			)

			continue
		}

		kept = append(kept, route)
	}

	return kept
}

// recordConflict - Reports a route collision on the object that lost it.
//
// An Event rather than a status condition: the Ingress API has no condition
// list to write to, and this has to work the same for Ingress and HTTPRoute.
// `kubectl describe` shows it either way, which is where someone looks when a
// route they created is not serving.
func (c *Controller) recordConflict(conflict routeConflict) {
	if c.events == nil {
		return
	}

	// Opaque on the route, since server/router knows nothing about Kubernetes.
	ref, ok := conflict.Loser.ObjectRef.(runtime.Object)
	if !ok || ref == nil {
		return
	}

	c.events.Eventf(ref, corev1.EventTypeWarning, "RouteConflict",
		"%s%s is already served by %s; this route is ignored",
		conflict.Host, conflict.Path, conflict.Winner.Source)
}
