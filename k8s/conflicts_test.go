//go:build all || unit
// +build all unit

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

func conflictingRoute(source string, host string, path string, age time.Duration) *router.Route {
	return &router.Route{
		ID:                source + path,
		Source:            source,
		Host:              host,
		Path:              path,
		PathType:          router.PathPrefix,
		CreationTimestamp: time.Now().Add(-age),
	}
}

// Nothing in the Ingress API expresses hostname ownership, so two namespaces
// can claim one host and path. Only one wins, and silently dropping the other
// leaves its author with a route that does not work and no way to find out
// why.
func TestFindRouteConflictsNamesWinnerAndLoser(t *testing.T) {
	older := conflictingRoute("Ingress victim-prod/site", "www.example.com", "/", 2*time.Hour)
	newer := conflictingRoute("Ingress tenant-a/squat", "www.example.com", "/", time.Minute)

	conflicts := findRouteConflicts([]*router.Route{newer, older})

	assert.Len(t, conflicts, 1)
	assert.Equal(t, "Ingress victim-prod/site", conflicts[0].Winner.Source, "the older object wins the tie")
	assert.Equal(t, "Ingress tenant-a/squat", conflicts[0].Loser.Source)
	assert.Contains(t, conflicts[0].String(), "www.example.com/")
}

// Different paths on one host are not a conflict; that is ordinary fanout.
func TestFindRouteConflictsIgnoresDistinctPaths(t *testing.T) {
	conflicts := findRouteConflicts([]*router.Route{
		conflictingRoute("Ingress a/one", "www.example.com", "/api", time.Hour),
		conflictingRoute("Ingress b/two", "www.example.com", "/assets", time.Hour),
	})

	assert.Empty(t, conflicts)
}

// One object legitimately produces several routes for one host and path (one
// per hostname or match); that is not a conflict with itself.
func TestFindRouteConflictsIgnoresOneObjectWithItself(t *testing.T) {
	conflicts := findRouteConflicts([]*router.Route{
		conflictingRoute("Ingress a/one", "www.example.com", "/", time.Hour),
		conflictingRoute("Ingress a/one", "www.example.com", "/", time.Hour),
	})

	assert.Empty(t, conflicts)
}

// A hostless route serves every hostname in the cluster, which is fine for one
// tenant and not fine for many.
func TestDropCatchAllRoutes(t *testing.T) {
	kept := dropCatchAllRoutes([]*router.Route{
		conflictingRoute("Ingress a/fallback", "", "/", time.Hour),
		conflictingRoute("Ingress b/site", "www.example.com", "/", time.Hour),
	})

	assert.Len(t, kept, 1)
	assert.Equal(t, "www.example.com", kept[0].Host)
}

func TestCertificateOwnerReportsASecondClaim(t *testing.T) {
	owners := make(certificateOwner)

	owners.claim("www.example.com", "Ingress a/one")
	owners.claim("www.example.com", "Ingress b/two")

	// Last writer wins, which is what the certificate map itself does; the
	// point is that it no longer happens silently.
	assert.Equal(t, "Ingress b/two", owners["www.example.com"])
}
