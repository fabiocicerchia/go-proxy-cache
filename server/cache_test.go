package server_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server"
)

func TestGetTTLFromWithMaxage(t *testing.T) {
	value := server.GetTTLFrom("max-age", `public, max-age=3600, s-maxage=86400`)
	assert.Equal(t, time.Duration(3600*time.Second), value)

	value = server.GetTTLFrom("max-age", `public,max-age=3600,s-maxage=86400`)
	assert.Equal(t, time.Duration(3600*time.Second), value)

	value = server.GetTTLFrom("max-age", `public, s-maxage=86400, max-age=3600`)
	assert.Equal(t, time.Duration(3600*time.Second), value)

	value = server.GetTTLFrom("max-age", `public,s-maxage=86400,max-age=3600`)
	assert.Equal(t, time.Duration(3600*time.Second), value)

	value = server.GetTTLFrom("max-age", `no-cache, max-age=0`)
	assert.Equal(t, time.Duration(0*time.Second), value)
}

func TestGetTTLFromWithSmaxage(t *testing.T) {
	value := server.GetTTLFrom("s-maxage", `public, max-age=3600, s-maxage=86400`)
	assert.Equal(t, time.Duration(86400*time.Second), value)

	value = server.GetTTLFrom("s-maxage", `public,max-age=3600,s-maxage=86400`)
	assert.Equal(t, time.Duration(86400*time.Second), value)

	value = server.GetTTLFrom("s-maxage", `public, s-maxage=86400, max-age=3600`)
	assert.Equal(t, time.Duration(86400*time.Second), value)

	value = server.GetTTLFrom("s-maxage", `public,s-maxage=86400,max-age=3600`)
	assert.Equal(t, time.Duration(86400*time.Second), value)

	value = server.GetTTLFrom("s-maxage", `public,max-age=3600`)
	assert.Equal(t, time.Duration(0*time.Second), value)

	value = server.GetTTLFrom("s-maxage", `no-cache, max-age=0`)
	assert.Equal(t, time.Duration(0*time.Second), value)
}

func TestGetTTLWhenNotSet(t *testing.T) {
	headers := map[string]interface{}{}
	value := server.GetTTL(headers)
	assert.Equal(t, time.Duration(0*time.Second), value)
}

func TestGetTTLWhenSet(t *testing.T) {
	headers := map[string]interface{}{
		"Cache-Control": "public, max-age=3600, s-maxage=86400",
	}
	value := server.GetTTL(headers)
	assert.Equal(t, time.Duration(86400*time.Second), value)
}
