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
	"net/http"
	"net/url"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

// TestFullPageRoundTripOnRedis drives StoreFullPage, RetrieveFullPage and
// PurgeFullPage against a real Redis, covering the Vary metadata lookup and
// the fresh/stale key pair.
func TestFullPageRoundTripOnRedis(t *testing.T) {
	ctx := context.Background()
	domainID := "full-page-roundtrip-test"
	circuit_breaker.InitCircuitBreaker(domainID, circuit_breaker.CircuitBreaker{Threshold: 2, FailureRate: 0.5}, log.StandardLogger())
	engine.InitConn(domainID, config.Cache{Hosts: []string{utils.GetEnv("REDIS_HOSTS", "localhost:6379")}}, log.StandardLogger())
	conn := engine.GetConn(domainID)
	_, err := conn.PurgeAll()
	assert.Nil(t, err)

	u, err := url.Parse("https://roundtrip.example/page?q=1")
	assert.Nil(t, err)

	newObj := func(method, lang string) cache.Object {
		return cache.Object{
			DomainID:        domainID,
			AllowedStatuses: []int{http.StatusOK},
			AllowedMethods:  []string{http.MethodGet},
			CurrentURIObject: cache.URIObj{
				URL:            *u,
				Method:         method,
				RequestHeaders: http.Header{"Accept-Language": []string{lang}},
			},
		}
	}

	stored := newObj(http.MethodGet, "en")
	stored.CurrentURIObject.StatusCode = http.StatusOK
	stored.CurrentURIObject.ResponseHeaders = http.Header{"Vary": []string{"Accept-Language"}, "Content-Type": []string{"text/plain"}}
	stored.CurrentURIObject.Content = [][]byte{[]byte("hello "), []byte("world")}

	done, err := stored.StoreFullPage(ctx, time.Minute)
	assert.Nil(t, err)
	assert.True(t, done)

	hit := newObj(http.MethodGet, "en")
	assert.Nil(t, hit.RetrieveFullPage())
	assert.False(t, hit.CurrentURIObject.Stale)
	assert.Equal(t, http.StatusOK, hit.CurrentURIObject.StatusCode)
	assert.Equal(t, "text/plain", hit.CurrentURIObject.ResponseHeaders.Get("Content-Type"))
	assert.Equal(t, stored.CurrentURIObject.Content, hit.CurrentURIObject.Content)

	// A different value of a Vary header is a different variant.
	otherLang := newObj(http.MethodGet, "it")
	assert.ErrorIs(t, otherLang.RetrieveFullPage(), cache.ErrEmptyValue)

	// Once the hard (fresh) key is gone the soft key is served as stale.
	meta, err := cache.FetchMetadata(domainID, http.MethodGet, *u, "")
	assert.Nil(t, err)
	assert.Equal(t, []string{"Accept-Language"}, meta)
	assert.Nil(t, conn.Del(ctx, cache.StorageKey(hit.CurrentURIObject, meta, hit.Variant)+cache.FreshSuffix))

	stale := newObj(http.MethodGet, "en")
	assert.Nil(t, stale.RetrieveFullPage())
	assert.True(t, stale.CurrentURIObject.Stale)
	assert.Equal(t, stored.CurrentURIObject.Content, stale.CurrentURIObject.Content)

	// PURGE removes every variant and its metadata.
	purged, err := newObj("PURGE", "").PurgeFullPage(ctx)
	assert.Nil(t, err)
	assert.True(t, purged)

	meta, err = cache.FetchMetadata(domainID, http.MethodGet, *u, "")
	assert.Nil(t, err)
	assert.Empty(t, meta)

	gone := newObj(http.MethodGet, "en")
	assert.ErrorIs(t, gone.RetrieveFullPage(), cache.ErrEmptyValue)
}
