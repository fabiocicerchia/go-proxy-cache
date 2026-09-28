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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
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
