//go:build all || functional
// +build all functional

package cache_test

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
	"net/url"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

func TestPurgeHostDeletesOnlyThatHost(t *testing.T) {
	ctx := context.Background()
	domainID := "purge-host-test"
	circuit_breaker.InitCircuitBreaker(domainID, circuit_breaker.CircuitBreaker{Threshold: 2, FailureRate: 0.5}, log.StandardLogger())
	engine.InitConn(domainID, config.Cache{Hosts: []string{utils.GetEnv("REDIS_HOSTS", "localhost:6379")}}, log.StandardLogger())
	conn := engine.GetConn(domainID)
	_, err := conn.PurgeAll()
	assert.Nil(t, err)

	sep := utils.StringSeparatorOne
	purged := []string{
		"DATA" + sep + "GET" + sep + "https://a.example/" + sep + "sum",
		"DATA" + sep + "HEAD" + sep + "https://a.example/x?y=1" + sep + "sum" + cache.FreshSuffix,
		"META" + sep + "GET" + sep + "https://a.example/x",
	}
	kept := []string{
		// http can be a separate domain with its own Redis, so it is not this
		// purge's to touch.
		"DATA" + sep + "GET" + sep + "http://a.example/" + sep + "sum",
		"DATA" + sep + "GET" + sep + "https://b.example/" + sep + "sum",
		"META" + sep + "GET" + sep + "https://a.example.b.example/",
		// A "*" in the method position would reach this one through its path.
		"DATA" + sep + "GET" + sep + "https://b.example/" + sep + "GET" + sep + "https://a.example/" + sep + "sum",
	}
	for _, k := range append(purged, kept...) {
		_, err := conn.Set(ctx, k, "v", 0)
		assert.Nil(t, err)
	}

	// Parsed the way the server parses a request-target, which is what decides
	// between Path and RawPath.
	purge := func(target string) (bool, error) {
		u, err := url.ParseRequestURI(target)
		assert.Nil(t, err)
		u.Scheme, u.Host = "https", "a.example"
		obj := cache.Object{
			DomainID:         domainID,
			AllowedMethods:   []string{"HEAD", "GET"},
			CurrentURIObject: cache.URIObj{URL: *u},
		}
		return storage.PurgeCachedContent(ctx, config.Upstream{}, storage.RequestCallDTO{CacheObject: obj})
	}

	// /%2A decodes to /* but names a single resource, so it purges nothing here.
	done, err := purge("/%2A")
	assert.Nil(t, err)
	assert.False(t, done, "PURGE /%2A must not purge the whole host")

	done, err = purge(cache.PurgeAllPath)
	assert.Nil(t, err)
	assert.True(t, done)

	for _, k := range purged {
		v, _ := conn.Get(k)
		assert.Empty(t, v, k)
	}
	for _, k := range kept {
		v, _ := conn.Get(k)
		assert.Equal(t, "v", v, k)
	}

	done, err = purge(cache.PurgeAllPath)
	assert.Nil(t, err)
	assert.False(t, done, "nothing left to purge reports not done, like a per-URL miss")
}
