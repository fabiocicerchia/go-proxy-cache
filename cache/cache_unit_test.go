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
