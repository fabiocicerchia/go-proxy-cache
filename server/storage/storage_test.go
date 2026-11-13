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

// --- StorageTTL (override_ttl)

func TestStorageTTLOverride(t *testing.T) {
	expires := time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)

	cases := []struct {
		name    string
		status  int
		headers http.Header
		cfg     config.Cache
		want    time.Duration
	}{
		{"override replaces max-age", 200, http.Header{"Cache-Control": []string{"max-age=60"}}, config.Cache{TTL: 10, OverrideTTL: 300}, 300 * time.Second},
		{"override replaces s-maxage", 200, http.Header{"Cache-Control": []string{"s-maxage=60"}}, config.Cache{OverrideTTL: 300}, 300 * time.Second},
		{"override replaces Expires", 200, http.Header{"Expires": []string{expires}}, config.Cache{OverrideTTL: 300}, 300 * time.Second},
		{"override applies when origin sends nothing", 200, http.Header{}, config.Cache{TTL: 10, OverrideTTL: 300}, 300 * time.Second},
		{"override applies over max-age=0", 200, http.Header{"Cache-Control": []string{"max-age=0"}}, config.Cache{OverrideTTL: 300}, 300 * time.Second},
		{"override off keeps max-age", 200, http.Header{"Cache-Control": []string{"max-age=60"}}, config.Cache{TTL: 10}, 60 * time.Second},
		{"override off falls back to ttl", 200, http.Header{}, config.Cache{TTL: 10}, 10 * time.Second},
		{"negative_ttl wins for its status", 404, http.Header{"Cache-Control": []string{"max-age=60"}}, config.Cache{OverrideTTL: 300, NegativeTTL: map[int]int{404: 30}}, 30 * time.Second},
		{"override applies to statuses negative_ttl doesn't list", 502, http.Header{}, config.Cache{OverrideTTL: 300, NegativeTTL: map[int]int{404: 30}}, 300 * time.Second},
	}

	for _, tc := range cases {
		uri := cache.URIObj{StatusCode: tc.status, ResponseHeaders: tc.headers}
		assert.Equal(t, tc.want, storage.StorageTTL(uri, tc.cfg), tc.name)
	}
}

func TestStoreGeneratedPageOverrideTTLDoesNotStoreForbidden(t *testing.T) {
	cases := []struct {
		name     string
		request  http.Header
		response http.Header
	}{
		{"private", http.Header{}, http.Header{"Cache-Control": []string{"private, max-age=60"}}},
		{"no-store", http.Header{}, http.Header{"Cache-Control": []string{"no-store"}}},
		{"no-cache", http.Header{}, http.Header{"Cache-Control": []string{"no-cache"}}},
		{"authorization without public/s-maxage", http.Header{"Authorization": []string{"Bearer x"}}, http.Header{"Cache-Control": []string{"max-age=60"}}},
		{"authorization without any header", http.Header{"Authorization": []string{"Bearer x"}}, http.Header{}},
	}

	for _, tc := range cases {
		rc := storage.RequestCallDTO{
			CacheObject: cache.Object{
				AllowedStatuses: []int{http.StatusOK},
				AllowedMethods:  []string{http.MethodGet},
				CurrentURIObject: cache.URIObj{
					Method:          http.MethodGet,
					StatusCode:      http.StatusOK,
					RequestHeaders:  tc.request,
					ResponseHeaders: tc.response,
					Content:         [][]byte{[]byte("hello")},
				},
			},
		}

		// No Redis connection is configured: had override_ttl made this
		// storable, storing would fail on the engine instead of false/nil.
		stored, err := storage.StoreGeneratedPage(context.Background(), rc, config.Cache{OverrideTTL: 300})

		assert.False(t, stored, tc.name)
		assert.NoError(t, err, tc.name)
	}
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
