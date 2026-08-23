package config

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"sync/atomic"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// Snapshot - An immutable view of the configuration.
//
// The proxy used to read the mutable package-level `Config` on every request
// and memoize domain lookups into a map guarded by a mutex. That works only
// because nothing ever writes the configuration after boot. The Kubernetes
// ingress controller does exactly that (routes appear and disappear as
// Ingress objects change), so the configuration is now published as an
// immutable snapshot swapped atomically: readers never take a lock and never
// observe a half-written value.
type Snapshot struct {
	Global  Configuration
	Domains Domains

	// lookup - Pre-computed "host@@scheme" and "host@@*" index, built once at
	// snapshot creation. This replaces the previous lazily-populated
	// domainsCache: because the snapshot is immutable there is nothing to
	// guard, and a lookup is a single map read instead of a linear scan over
	// every configured domain.
	lookup map[string]Configuration
}

// currentSnapshot - The configuration currently serving traffic.
var currentSnapshot atomic.Pointer[Snapshot]

// NewSnapshot - Builds an immutable snapshot and its lookup index.
func NewSnapshot(global Configuration, domains Domains) *Snapshot {
	s := &Snapshot{
		Global:  global,
		Domains: domains,
		lookup:  make(map[string]Configuration, len(domains)*2+2),
	}

	// Least specific first, so the more specific entries below overwrite them.

	// Third round equivalent: the global upstream host, any scheme.
	if global.Server.Upstream.Host != "" {
		s.lookup[key(global.Server.Upstream.Host, SchemeWildcard)] = global
	}

	// Second round equivalent: per-domain host, any scheme.
	for _, v := range domains {
		if v.Server.Upstream.Host == "" {
			continue
		}

		s.lookup[key(v.Server.Upstream.Host, SchemeWildcard)] = v
	}

	// First round equivalent: per-domain host and scheme.
	for _, v := range domains {
		if v.Server.Upstream.Host == "" {
			continue
		}

		s.lookup[key(v.Server.Upstream.Host, v.Server.Upstream.Scheme)] = v
	}

	return s
}

func key(host string, scheme string) string {
	return host + utils.StringSeparatorOne + scheme
}

// Current - Returns the snapshot currently serving traffic.
//
// Falls back to a snapshot built from the package-level Config when nothing
// has been published yet: tests (and any code path that mutates config.Config
// directly) must keep working without an explicit Publish call.
func Current() *Snapshot {
	if s := currentSnapshot.Load(); s != nil {
		return s
	}

	return NewSnapshot(Config, Config.Domains)
}

// Publish - Atomically swaps the configuration serving traffic.
func Publish(s *Snapshot) {
	currentSnapshot.Store(s)
}

// PublishFromConfig - Publishes a snapshot built from the package-level Config.
func PublishFromConfig() {
	Publish(NewSnapshot(Config, Config.Domains))
}

// Reset - Drops the published snapshot, so Current() falls back to Config.
// Only meant for tests that mutate config.Config directly.
func Reset() {
	currentSnapshot.Store(nil)
}

// DomainConf - Returns the configuration for the requested domain.
func (s *Snapshot) DomainConf(domain string, scheme string) (Configuration, bool) {
	host := utils.StripPort(domain)

	if conf, ok := s.lookup[key(host, scheme)]; ok {
		return conf, true
	}

	if conf, ok := s.lookup[key(host, SchemeWildcard)]; ok {
		return conf, true
	}

	return Configuration{}, false
}
