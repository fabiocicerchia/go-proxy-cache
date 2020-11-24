// +build unit

package utils_test

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

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetTTLFromCacheControlWithMaxage(t *testing.T) {
	value := utils.GetTTLFromCacheControl("max-age", `public, max-age=3600, s-maxage=86400`)
	assert.Equal(t, 3600*time.Second, value)

	value = utils.GetTTLFromCacheControl("max-age", `public,max-age=3600,s-maxage=86400`)
	assert.Equal(t, 3600*time.Second, value)

	value = utils.GetTTLFromCacheControl("max-age", `public, s-maxage=86400, max-age=3600`)
	assert.Equal(t, 3600*time.Second, value)

	value = utils.GetTTLFromCacheControl("max-age", `public,s-maxage=86400,max-age=3600`)
	assert.Equal(t, 3600*time.Second, value)

	value = utils.GetTTLFromCacheControl("max-age", `no-cache, max-age=0`)
	assert.Equal(t, 0*time.Second, value)
}

func TestGetTTLFromCacheControlWithSmaxage(t *testing.T) {
	value := utils.GetTTLFromCacheControl("s-maxage", `public, max-age=3600, s-maxage=86400`)
	assert.Equal(t, 86400*time.Second, value)

	value = utils.GetTTLFromCacheControl("s-maxage", `public,max-age=3600,s-maxage=86400`)
	assert.Equal(t, 86400*time.Second, value)

	value = utils.GetTTLFromCacheControl("s-maxage", `public, s-maxage=86400, max-age=3600`)
	assert.Equal(t, 86400*time.Second, value)

	value = utils.GetTTLFromCacheControl("s-maxage", `public,s-maxage=86400,max-age=3600`)
	assert.Equal(t, 86400*time.Second, value)

	value = utils.GetTTLFromCacheControl("s-maxage", `public,max-age=3600`)
	assert.Equal(t, 0*time.Second, value)

	value = utils.GetTTLFromCacheControl("s-maxage", `no-cache, max-age=0`)
	assert.Equal(t, 0*time.Second, value)
}

func TestGetTTLWhenNotSet(t *testing.T) {
	headers := http.Header{}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, 1*time.Second, value)
}

func TestGetTTLWhenSetCacheControl(t *testing.T) {
	headers := http.Header{
		"Cache-Control": []string{"public, max-age=3600, s-maxage=86400"},
	}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, 86400*time.Second, value)
}

func TestGetTTLWhenCacheControlNoCache(t *testing.T) {
	headers := http.Header{
		"Cache-Control": []string{"private, no-cache, max-age=3600"},
	}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, 0*time.Second, value)
}

func TestGetTTLWhenCacheControlNoStore(t *testing.T) {
	headers := http.Header{
		"Cache-Control": []string{"private, no-store, max-age=3600"},
	}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, 0*time.Second, value)
}

func TestGetTTLWhenSetExpires(t *testing.T) {
	expireDate := time.Now().UTC().Add(60 * time.Second)
	expires := expireDate.Format(http.TimeFormat)

	headers := http.Header{
		"Expires": []string{expires},
	}
	value := utils.GetTTL(headers, 1)
	assert.Less(t, float64(59), value.Seconds())
	assert.Greater(t, float64(60), value.Seconds())
}

func TestGetTTLWhenSetCacheControlAndExpires(t *testing.T) {
	expireDate := time.Now().Add(60 * time.Second)
	expires := expireDate.Format(http.TimeFormat)

	headers := http.Header{
		"Cache-Control": []string{"public, max-age=3600, s-maxage=86400"},
		"Expires":       []string{expires},
	}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, 86400*time.Second, value)
}
