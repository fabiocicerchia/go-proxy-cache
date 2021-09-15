//go:build all || unit
// +build all unit

package ttl_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/utils/ttl"
)

func TestGetTTLFromCacheControlWithMaxage(t *testing.T) {
	value := ttl.GetTTLFromCacheControl("max-age", `public, max-age=3600, s-maxage=86400`)
	assert.Equal(t, 3600*time.Second, value)

	value = ttl.GetTTLFromCacheControl("max-age", `public,max-age=3600,s-maxage=86400`)
	assert.Equal(t, 3600*time.Second, value)

	value = ttl.GetTTLFromCacheControl("max-age", `public, s-maxage=86400, max-age=3600`)
	assert.Equal(t, 3600*time.Second, value)

	value = ttl.GetTTLFromCacheControl("max-age", `public,s-maxage=86400,max-age=3600`)
	assert.Equal(t, 3600*time.Second, value)

	value = ttl.GetTTLFromCacheControl("max-age", `no-cache, max-age=0`)
	assert.Equal(t, 0*time.Second, value)
}

func TestGetTTLFromCacheControlWithSmaxage(t *testing.T) {
	value := ttl.GetTTLFromCacheControl("s-maxage", `public, max-age=3600, s-maxage=86400`)
	assert.Equal(t, 86400*time.Second, value)

	value = ttl.GetTTLFromCacheControl("s-maxage", `public,max-age=3600,s-maxage=86400`)
	assert.Equal(t, 86400*time.Second, value)

	value = ttl.GetTTLFromCacheControl("s-maxage", `public, s-maxage=86400, max-age=3600`)
	assert.Equal(t, 86400*time.Second, value)

	value = ttl.GetTTLFromCacheControl("s-maxage", `public,s-maxage=86400,max-age=3600`)
	assert.Equal(t, 86400*time.Second, value)

	value = ttl.GetTTLFromCacheControl("s-maxage", `public,max-age=3600`)
	assert.Equal(t, 0*time.Second, value)

	value = ttl.GetTTLFromCacheControl("s-maxage", `no-cache, max-age=0`)
	assert.Equal(t, 0*time.Second, value)
}

func TestGetTTLWhenNotSet(t *testing.T) {
	headers := http.Header{}
	value := ttl.GetTTL(headers, 1)
	assert.Equal(t, 1*time.Second, value)
}

func TestGetTTLWhenSetCacheControlSmaxage(t *testing.T) {
	headers := http.Header{
		"Cache-Control": []string{"public, max-age=3600, s-maxage=86400"},
	}
	value := ttl.GetTTL(headers, 1)
	assert.Equal(t, 86400*time.Second, value)
}

func TestGetTTLWhenSetCacheControlMaxage(t *testing.T) {
	headers := http.Header{
		"Cache-Control": []string{"public, max-age=3600"},
	}
	value := ttl.GetTTL(headers, 1)
	assert.Equal(t, 3600*time.Second, value)
}

func TestGetTTLWhenSetCacheControlEmpty(t *testing.T) {
	headers := http.Header{
		"Cache-Control": []string{"public"},
	}
	value := ttl.GetTTL(headers, 1)
	assert.Equal(t, 1*time.Second, value)
}

func TestGetTTLWhenCacheControlNoCache(t *testing.T) {
	headers := http.Header{
		"Cache-Control": []string{"private, no-cache, max-age=3600"},
	}
	value := ttl.GetTTL(headers, 1)
	assert.Equal(t, 0*time.Second, value)
}

func TestGetTTLWhenCacheControlNoStore(t *testing.T) {
	headers := http.Header{
		"Cache-Control": []string{"private, no-store, max-age=3600"},
	}
	value := ttl.GetTTL(headers, 1)
	assert.Equal(t, 0*time.Second, value)
}

func TestGetTTLWhenSetExpires(t *testing.T) {
	expireDate := time.Now().UTC().Add(60 * time.Second)
	expires := expireDate.Format(http.TimeFormat)

	headers := http.Header{
		"Expires": []string{expires},
	}
	value := ttl.GetTTL(headers, 1)
	assert.Less(t, 59.0, value.Seconds())
	assert.Greater(t, 60.0, value.Seconds())
}

func TestGetTTLWhenInvalidExpires(t *testing.T) {
	expires := "test"

	headers := http.Header{
		"Expires": []string{expires},
	}
	value := ttl.GetTTL(headers, 1)
	assert.Equal(t, 1.0, value.Seconds())
}

func TestGetTTLWhenSetCacheControlAndExpires(t *testing.T) {
	expireDate := time.Now().Add(60 * time.Second)
	expires := expireDate.Format(http.TimeFormat)

	headers := http.Header{
		"Cache-Control": []string{"public, max-age=3600, s-maxage=86400"},
		"Expires":       []string{expires},
	}
	value := ttl.GetTTL(headers, 1)
	assert.Equal(t, 86400*time.Second, value)
}
