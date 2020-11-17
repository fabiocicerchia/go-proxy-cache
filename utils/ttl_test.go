// +build unit

package utils_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetTTLFromCacheControlWithMaxage(t *testing.T) {
	value := utils.GetTTLFromCacheControl("max-age", `public, max-age=3600, s-maxage=86400`)
	assert.Equal(t, time.Duration(3600*time.Second), value)

	value = utils.GetTTLFromCacheControl("max-age", `public,max-age=3600,s-maxage=86400`)
	assert.Equal(t, time.Duration(3600*time.Second), value)

	value = utils.GetTTLFromCacheControl("max-age", `public, s-maxage=86400, max-age=3600`)
	assert.Equal(t, time.Duration(3600*time.Second), value)

	value = utils.GetTTLFromCacheControl("max-age", `public,s-maxage=86400,max-age=3600`)
	assert.Equal(t, time.Duration(3600*time.Second), value)

	value = utils.GetTTLFromCacheControl("max-age", `no-cache, max-age=0`)
	assert.Equal(t, time.Duration(0*time.Second), value)
}

func TestGetTTLFromCacheControlWithSmaxage(t *testing.T) {
	value := utils.GetTTLFromCacheControl("s-maxage", `public, max-age=3600, s-maxage=86400`)
	assert.Equal(t, time.Duration(86400*time.Second), value)

	value = utils.GetTTLFromCacheControl("s-maxage", `public,max-age=3600,s-maxage=86400`)
	assert.Equal(t, time.Duration(86400*time.Second), value)

	value = utils.GetTTLFromCacheControl("s-maxage", `public, s-maxage=86400, max-age=3600`)
	assert.Equal(t, time.Duration(86400*time.Second), value)

	value = utils.GetTTLFromCacheControl("s-maxage", `public,s-maxage=86400,max-age=3600`)
	assert.Equal(t, time.Duration(86400*time.Second), value)

	value = utils.GetTTLFromCacheControl("s-maxage", `public,max-age=3600`)
	assert.Equal(t, time.Duration(0*time.Second), value)

	value = utils.GetTTLFromCacheControl("s-maxage", `no-cache, max-age=0`)
	assert.Equal(t, time.Duration(0*time.Second), value)
}

func TestGetTTLWhenNotSet(t *testing.T) {
	headers := map[string]interface{}{}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, time.Duration(1*time.Second), value)
}

func TestGetTTLWhenSetCacheControl(t *testing.T) {
	headers := map[string]interface{}{
		"Cache-Control": "public, max-age=3600, s-maxage=86400",
	}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, time.Duration(86400*time.Second), value)
}

func TestGetTTLWhenCacheControlNoCache(t *testing.T) {
	headers := map[string]interface{}{
		"Cache-Control": "private, no-cache, max-age=3600",
	}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, time.Duration(0*time.Second), value)
}

func TestGetTTLWhenCacheControlNoStore(t *testing.T) {
	headers := map[string]interface{}{
		"Cache-Control": "private, no-store, max-age=3600",
	}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, time.Duration(0*time.Second), value)
}

func TestGetTTLWhenSetExpires(t *testing.T) {
	expireDate := time.Now().UTC().Add(60 * time.Second)
	expires := expireDate.Format(http.TimeFormat)

	headers := map[string]interface{}{
		"Expires": expires,
	}
	value := utils.GetTTL(headers, 1)
	assert.Less(t, float64(59), value.Seconds())
	assert.Greater(t, float64(60), value.Seconds())
}

func TestGetTTLWhenSetCacheControlAndExpires(t *testing.T) {
	expireDate := time.Now().Add(60 * time.Second)
	expires := expireDate.Format(http.TimeFormat)

	headers := map[string]interface{}{
		"Cache-Control": "public, max-age=3600, s-maxage=86400",
		"Expires":       expires,
	}
	value := utils.GetTTL(headers, 1)
	assert.Equal(t, time.Duration(86400*time.Second), value)
}
