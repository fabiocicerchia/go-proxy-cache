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
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
)

func TestHasRangeRequest(t *testing.T) {
	withRange := cache.URIObj{RequestHeaders: http.Header{"Range": []string{"bytes=0-99"}}}
	assert.True(t, withRange.HasRangeRequest())

	withoutRange := cache.URIObj{RequestHeaders: http.Header{}}
	assert.False(t, withoutRange.HasRangeRequest())
}

func TestIsPartialContent(t *testing.T) {
	statusPartial := cache.URIObj{StatusCode: http.StatusPartialContent, ResponseHeaders: http.Header{}}
	assert.True(t, statusPartial.IsPartialContent())

	contentRange := cache.URIObj{StatusCode: http.StatusOK, ResponseHeaders: http.Header{"Content-Range": []string{"bytes 0-99/200"}}}
	assert.True(t, contentRange.IsPartialContent())

	full := cache.URIObj{StatusCode: http.StatusOK, ResponseHeaders: http.Header{}}
	assert.False(t, full.IsPartialContent())
}

func TestStoreFullPageBypassesCacheForRangeRequest(t *testing.T) {
	obj := cache.Object{
		AllowedStatuses: []int{http.StatusOK},
		AllowedMethods:  []string{http.MethodGet},
		CurrentURIObject: cache.URIObj{
			Method:          http.MethodGet,
			StatusCode:      http.StatusOK,
			RequestHeaders:  http.Header{"Range": []string{"bytes=0-99"}},
			ResponseHeaders: http.Header{},
			Content:         [][]byte{[]byte("hello")},
		},
	}

	// No Redis connection is configured; if the Range guard didn't short-circuit
	// before reaching the engine, this would fail with a connection error
	// instead of the expected "not stored" false/nil.
	stored, err := obj.StoreFullPage(context.Background(), time.Minute)

	assert.False(t, stored)
	assert.NoError(t, err)
}

func TestStoreFullPageBypassesCacheForPartialContentResponse(t *testing.T) {
	obj := cache.Object{
		AllowedStatuses: []int{http.StatusOK, http.StatusPartialContent},
		AllowedMethods:  []string{http.MethodGet},
		CurrentURIObject: cache.URIObj{
			Method:          http.MethodGet,
			StatusCode:      http.StatusPartialContent,
			RequestHeaders:  http.Header{},
			ResponseHeaders: http.Header{"Content-Range": []string{"bytes 0-99/200"}},
			Content:         [][]byte{[]byte("hello")},
		},
	}

	stored, err := obj.StoreFullPage(context.Background(), time.Minute)

	assert.False(t, stored)
	assert.NoError(t, err)
}

func TestRetrieveFullPageBypassesCacheForRangeRequest(t *testing.T) {
	obj := cache.Object{
		CurrentURIObject: cache.URIObj{
			Method:         http.MethodGet,
			RequestHeaders: http.Header{"Range": []string{"bytes=0-99"}},
		},
	}

	// No Redis connection is configured; if the Range guard didn't short-circuit
	// before FetchMetadata/engine.GetConn, this would fail with a connection
	// error instead of the expected ErrEmptyValue (treated as a cache miss).
	err := obj.RetrieveFullPage()

	assert.ErrorIs(t, err, cache.ErrEmptyValue)
}

func storableObject(requestHeaders, responseHeaders http.Header) cache.Object {
	return cache.Object{
		AllowedStatuses: []int{http.StatusOK},
		AllowedMethods:  []string{http.MethodGet},
		CurrentURIObject: cache.URIObj{
			Method:          http.MethodGet,
			StatusCode:      http.StatusOK,
			RequestHeaders:  requestHeaders,
			ResponseHeaders: responseHeaders,
			Content:         [][]byte{[]byte("hello")},
		},
	}
}

func TestIsStorableBySharedCache(t *testing.T) {
	cases := []struct {
		name     string
		request  http.Header
		response http.Header
		want     bool
	}{
		{"plain public response", http.Header{}, http.Header{"Cache-Control": []string{"public, max-age=60"}}, true},
		{"no cache headers", http.Header{}, http.Header{}, true},
		{"private", http.Header{}, http.Header{"Cache-Control": []string{"private, max-age=60"}}, false},
		{"qualified private", http.Header{}, http.Header{"Cache-Control": []string{`private="Set-Cookie"`}}, false},
		{"no-store", http.Header{}, http.Header{"Cache-Control": []string{"no-store"}}, false},
		{"no-cache", http.Header{}, http.Header{"Cache-Control": []string{"no-cache"}}, false},
		{"private in a second header", http.Header{}, http.Header{"Cache-Control": []string{"max-age=60", "private"}}, false},
		{"substring is not a directive", http.Header{}, http.Header{"Cache-Control": []string{"x-no-store-hint, max-age=60"}}, true},
		{"authorization without permission", http.Header{"Authorization": []string{"Bearer x"}}, http.Header{"Cache-Control": []string{"max-age=60"}}, false},
		{"authorization with public", http.Header{"Authorization": []string{"Bearer x"}}, http.Header{"Cache-Control": []string{"public, max-age=60"}}, true},
		{"authorization with s-maxage", http.Header{"Authorization": []string{"Bearer x"}}, http.Header{"Cache-Control": []string{"s-maxage=60"}}, true},
		{"authorization with must-revalidate", http.Header{"Authorization": []string{"Bearer x"}}, http.Header{"Cache-Control": []string{"must-revalidate, max-age=60"}}, true},
		{"authorization with public but private", http.Header{"Authorization": []string{"Bearer x"}}, http.Header{"Cache-Control": []string{"public, private"}}, false},
	}

	for _, tc := range cases {
		uri := cache.URIObj{RequestHeaders: tc.request, ResponseHeaders: tc.response}
		assert.Equal(t, tc.want, uri.IsStorableBySharedCache(), tc.name)
	}
}

func TestStoreFullPageBypassesCacheForPrivateResponse(t *testing.T) {
	obj := storableObject(http.Header{}, http.Header{"Cache-Control": []string{"private, max-age=60"}})

	// No Redis connection is configured: reaching the engine would error.
	stored, err := obj.StoreFullPage(context.Background(), time.Minute)

	assert.False(t, stored)
	assert.NoError(t, err)
}

func TestStoreFullPageBypassesCacheForAuthorizedRequest(t *testing.T) {
	obj := storableObject(http.Header{"Authorization": []string{"Bearer x"}}, http.Header{"Cache-Control": []string{"max-age=60"}})

	stored, err := obj.StoreFullPage(context.Background(), time.Minute)

	assert.False(t, stored)
	assert.NoError(t, err)
}
