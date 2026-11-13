//go:build all || unit
// +build all unit

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
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
)

func uriObj(path string, headers http.Header) cache.URIObj {
	return cache.URIObj{
		URL:            url.URL{Scheme: "http", Host: "example.com", Path: path},
		Method:         "GET",
		RequestHeaders: headers,
	}
}

// Two routes can differ only by a header match -- a canary on x-version is the
// usual case -- while sharing a method, URL and Vary checksum. Without the
// variant they share one entry and serve each other's bodies.
func TestStorageKeySeparatesRoutesThatDifferOnlyByMatch(t *testing.T) {
	obj := uriObj("/api", http.Header{"X-Version": []string{"canary"}})

	stable := cache.StorageKey(obj, []string{}, "httproute/default/demo/gw/0/0/0")
	canary := cache.StorageKey(obj, []string{}, "httproute/default/demo/gw/1/0/0")

	assert.NotEqual(t, stable, canary, "routes with distinct identities must not share a cache entry")
}

// Existing deployments must keep their keys: an empty variant contributes
// nothing at all, not an empty trailing segment.
func TestStorageKeyIsUnchangedWithoutAVariant(t *testing.T) {
	obj := uriObj("/api", http.Header{})

	withoutVariant := cache.StorageKey(obj, []string{}, "")
	sameAgain := cache.StorageKey(obj, []string{}, "")

	assert.Equal(t, withoutVariant, sameAgain)
	assert.NotContains(t, withoutVariant, "DATA@@@@", "an empty variant must not add a separator")

	withVariant := cache.StorageKey(obj, []string{}, "route-a")
	assert.NotEqual(t, withoutVariant, withVariant)
	assert.Equal(t, withoutVariant+"@@route-a", withVariant, "the variant is appended, so the prefix is preserved")
}

// The same route must resolve to the same key on store and on retrieve.
func TestStorageKeyIsStableForOneRoute(t *testing.T) {
	obj := uriObj("/api", http.Header{})

	first := cache.StorageKey(obj, []string{}, "route-a")
	second := cache.StorageKey(obj, []string{}, "route-a")

	assert.Equal(t, first, second)
}

// Vary is a property of the response, so two routes serving one method and URL
// can answer with different Vary headers. Sharing one metadata list makes each
// route compute its header checksum over the other's headers and miss for
// ever, while still writing entries nothing will read.
func TestMetadataKeySeparatesRoutes(t *testing.T) {
	target := url.URL{Scheme: "http", Host: "example.com", Path: "/api"}

	stable := cache.MetadataKeyForTest("GET", target, "httproute/default/demo/gw/0/0/0")
	canary := cache.MetadataKeyForTest("GET", target, "httproute/default/demo/gw/1/0/0")

	assert.NotEqual(t, stable, canary, "routes with distinct identities must not share metadata")
}

// Keys written before routes existed keep their shape.
func TestMetadataKeyIsUnchangedWithoutAVariant(t *testing.T) {
	target := url.URL{Scheme: "http", Host: "example.com", Path: "/api"}

	withoutVariant := cache.MetadataKeyForTest("GET", target, "")

	assert.Equal(t, "META@@GET@@http://example.com/api", withoutVariant)
	assert.Equal(t, withoutVariant+"@@route-a", cache.MetadataKeyForTest("GET", target, "route-a"))
}
