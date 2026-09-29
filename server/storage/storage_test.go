//go:build all || unit
// +build all unit

package storage_test

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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/utils/ttl"
)

// --- ApplyNegativeTTL

func TestApplyNegativeTTLWithOverride(t *testing.T) {
	value := storage.ApplyNegativeTTL(404, map[int]int{404: 30, 502: 10}, 3600*time.Second)

	assert.Equal(t, 30*time.Second, value)
}

func TestApplyNegativeTTLWithoutOverride(t *testing.T) {
	value := storage.ApplyNegativeTTL(200, map[int]int{404: 30}, 3600*time.Second)

	assert.Equal(t, 3600*time.Second, value)
}

func TestApplyNegativeTTLWithNilMap(t *testing.T) {
	value := storage.ApplyNegativeTTL(200, nil, 3600*time.Second)

	assert.Equal(t, 3600*time.Second, value)
}

// --- StoreGeneratedPage

func TestStoreGeneratedPageNegativeTTLDoesNotStorePrivate(t *testing.T) {
	rc := storage.RequestCallDTO{
		CacheObject: cache.Object{
			AllowedStatuses: []int{http.StatusNotFound},
			AllowedMethods:  []string{http.MethodGet},
			CurrentURIObject: cache.URIObj{
				Method:          http.MethodGet,
				StatusCode:      http.StatusNotFound,
				RequestHeaders:  http.Header{},
				ResponseHeaders: http.Header{"Cache-Control": []string{"private"}},
				Content:         [][]byte{[]byte("not found")},
			},
		},
	}

	// No Redis connection is configured: if the negative TTL turned this
	// private 404 into a positive TTL, storing would fail on the engine
	// instead of returning the expected "not stored" false/nil.
	stored, err := storage.StoreGeneratedPage(context.Background(), rc, config.Cache{NegativeTTL: map[int]int{404: 30}})

	assert.False(t, stored)
	assert.NoError(t, err)
}

// TestStoreGeneratedPageNegativeTTLFromYAML - negative_ttl set in the config
// file reaches the domain config the handler stores with, so a 404 with no
// cache headers gets its short TTL instead of being skipped.
func TestStoreGeneratedPageNegativeTTLFromYAML(t *testing.T) {
	orig := config.Config
	t.Cleanup(func() { config.Config = orig })

	file := filepath.Join(t.TempDir(), "config.yml")
	yaml := "cache:\n  negative_ttl:\n    404: 30\ndomains:\n  neg:\n    server:\n      upstream:\n        host: neg.example\n"
	assert.NoError(t, os.WriteFile(file, []byte(yaml), 0o600))
	config.InitConfigFromFileOrEnv(file)

	domainConfig, found := config.DomainConf("neg.example", "http")
	assert.True(t, found)

	// Same wiring as handler.ConvertToRequestCallDTO / storeResponse.
	rc := storage.RequestCallDTO{
		CacheObject: cache.Object{
			AllowedStatuses: domainConfig.Cache.EffectiveAllowedStatuses(),
			AllowedMethods:  domainConfig.Cache.AllowedMethods,
			DomainID:        domainConfig.Server.Upstream.GetDomainID(),
			CurrentURIObject: cache.URIObj{
				Method:          http.MethodGet,
				StatusCode:      http.StatusNotFound,
				RequestHeaders:  http.Header{},
				ResponseHeaders: http.Header{},
				Content:         [][]byte{[]byte("not found")},
			},
		},
	}

	headers, status := rc.CacheObject.CurrentURIObject.ResponseHeaders, rc.CacheObject.CurrentURIObject.StatusCode
	assert.Equal(t, 30*time.Second, storage.ApplyNegativeTTL(status, domainConfig.Cache.NegativeTTL, ttl.GetTTL(headers, domainConfig.Cache.TTL)))

	// No Redis here: reaching the engine proves the 404 was accepted with a
	// positive TTL. Without the negative TTL it is skipped as (false, nil).
	stored, err := storage.StoreGeneratedPage(context.Background(), rc, domainConfig.Cache)
	assert.False(t, stored)
	assert.ErrorContains(t, err, "missing redis connection")
}
