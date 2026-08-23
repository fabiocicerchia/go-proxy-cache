package server

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"sync"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

// Registry - Tracks which load balancers are currently live.
//
// In the static configuration the balancers are created once at boot and never
// touched again. Under the Kubernetes ingress controller they come and go with
// the routes, so they need an idempotent create and an explicit teardown that
// stops the health-check goroutine.
type Registry struct {
	mu    sync.Mutex
	known map[string]config.Upstream
}

// NewRegistry - Builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{known: make(map[string]config.Upstream)}
}

// EnsureBalancer - Creates or updates the balancer for an ID, skipping the work
// when the upstream is unchanged.
func (r *Registry) EnsureBalancer(id string, upstream config.Upstream) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if previous, ok := r.known[id]; ok && sameUpstream(previous, upstream) {
		return
	}

	balancer.Init(id, upstream)
	r.known[id] = upstream
}

// Reconcile - Makes the registry hold exactly the given set of balancers,
// removing any that are no longer wanted.
func (r *Registry) Reconcile(wanted map[string]config.Upstream) {
	for id, upstream := range wanted {
		r.EnsureBalancer(id, upstream)
	}

	r.mu.Lock()
	stale := make([]string, 0)

	for id := range r.known {
		if _, ok := wanted[id]; !ok {
			stale = append(stale, id)
		}
	}
	r.mu.Unlock()

	for _, id := range stale {
		r.Remove(id)
	}
}

// Remove - Tears down a balancer and forgets it.
func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.known[id]; !ok {
		return
	}

	balancer.Remove(id)
	delete(r.known, id)

	logger.GetGlobal().Debugf("Removed load balancer %s", id)
}

func sameUpstream(a config.Upstream, b config.Upstream) bool {
	if a.BalancingAlgorithm != b.BalancingAlgorithm ||
		a.Host != b.Host ||
		a.Scheme != b.Scheme ||
		a.Port != b.Port ||
		len(a.Endpoints) != len(b.Endpoints) {
		return false
	}

	for i := range a.Endpoints {
		if a.Endpoints[i] != b.Endpoints[i] {
			return false
		}
	}

	return sameHealthCheck(a.HealthCheck, b.HealthCheck)
}

func sameHealthCheck(a config.HealthCheck, b config.HealthCheck) bool {
	if a.Timeout != b.Timeout ||
		a.Interval != b.Interval ||
		a.Port != b.Port ||
		a.Scheme != b.Scheme ||
		a.AllowInsecure != b.AllowInsecure ||
		len(a.StatusCodes) != len(b.StatusCodes) {
		return false
	}

	for i := range a.StatusCodes {
		if a.StatusCodes[i] != b.StatusCodes[i] {
			return false
		}
	}

	return true
}
