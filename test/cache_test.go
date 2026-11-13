//go:build all || endtoend
// +build all endtoend

package test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const cacheJourneyURL = "http://testing.local:50080/push"

// TestCacheMissHitPurgeJourney drives the core journey against the local
// nginx upstream (no external sites): first request is a MISS, the repeat is
// served from cache as a HIT, PURGE evicts it and the next request is a MISS.
func TestCacheMissHitPurgeJourney(t *testing.T) {
	purgeCacheJourney(t) // start from an empty cache, whatever ran before
	defer purgeCacheJourney(t)

	status, cacheStatus, body := getCacheJourney(t)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "MISS", cacheStatus)
	assert.Equal(t, "push", body)

	status, cacheStatus, body = getCacheJourney(t)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "HIT", cacheStatus)
	assert.Equal(t, "push", body)

	assert.Equal(t, http.StatusOK, purgeCacheJourney(t))

	status, cacheStatus, body = getCacheJourney(t)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "MISS", cacheStatus)
	assert.Equal(t, "push", body)
}

func getCacheJourney(t *testing.T) (int, string, string) {
	req, err := http.NewRequest("GET", cacheJourneyURL, nil)
	assert.Nil(t, err)
	req.Host = "testing.local"

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	return res.StatusCode, res.Header.Get("X-Go-Proxy-Cache-Status"), string(body)
}

func purgeCacheJourney(t *testing.T) int {
	req, err := http.NewRequest("PURGE", cacheJourneyURL, nil)
	assert.Nil(t, err)
	req.Host = "testing.local"
	req.Header.Set("X-Go-Proxy-Cache-Purge-Key", "test-only-purge-key")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	return res.StatusCode
}
