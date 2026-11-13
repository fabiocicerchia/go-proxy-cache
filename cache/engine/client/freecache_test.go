//go:build all || unit
// +build all unit

package client

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache/

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

func TestFreecacheClientSetGet(t *testing.T) {
	fc := NewFreecacheClient(config.Cache{})

	ok, err := fc.Set(context.Background(), "k1", "v1", time.Minute)
	assert.True(t, ok)
	assert.NoError(t, err)

	value, err := fc.Get("k1")
	assert.NoError(t, err)
	assert.Equal(t, "v1", value)
}

func TestFreecacheClientGetMissReturnsEmptyNoError(t *testing.T) {
	fc := NewFreecacheClient(config.Cache{})

	value, err := fc.Get("missing")
	assert.NoError(t, err)
	assert.Equal(t, "", value)
}

func TestFreecacheClientDel(t *testing.T) {
	fc := NewFreecacheClient(config.Cache{})

	_, _ = fc.Set(context.Background(), "k1", "v1", time.Minute)
	assert.NoError(t, fc.Del(context.Background(), "k1"))

	value, err := fc.Get("k1")
	assert.NoError(t, err)
	assert.Equal(t, "", value)
}

func TestFreecacheClientDelWildcard(t *testing.T) {
	fc := NewFreecacheClient(config.Cache{})

	_, _ = fc.Set(context.Background(), "DATA|GET|foo", "v1", time.Minute)
	_, _ = fc.Set(context.Background(), "DATA|GET|bar", "v2", time.Minute)
	_, _ = fc.Set(context.Background(), "META|GET|foo", "v3", time.Minute)

	affected, err := fc.DelWildcard(context.Background(), "DATA|*")
	assert.NoError(t, err)
	assert.Equal(t, 2, affected)

	value, _ := fc.Get("META|GET|foo")
	assert.Equal(t, "v3", value, "non-matching key must survive")
}

func TestFreecacheClientPushAndList(t *testing.T) {
	fc := NewFreecacheClient(config.Cache{})

	assert.NoError(t, fc.Push(context.Background(), "list1", []string{"a", "b"}))
	assert.NoError(t, fc.Push(context.Background(), "list1", []string{"c"}))

	values, err := fc.List("list1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, values)
}

func TestFreecacheClientListMissingReturnsEmpty(t *testing.T) {
	fc := NewFreecacheClient(config.Cache{})

	values, err := fc.List("missing")
	assert.NoError(t, err)
	assert.Equal(t, []string{}, values)
}

func TestFreecacheClientEncodeDecodeRoundTrip(t *testing.T) {
	fc := NewFreecacheClient(config.Cache{})

	type payload struct{ Name string }

	encoded, err := fc.Encode(payload{Name: "test"})
	assert.NoError(t, err)

	var out payload
	assert.NoError(t, fc.Decode(encoded, &out))
	assert.Equal(t, "test", out.Name)
}

func TestFreecacheClientPing(t *testing.T) {
	fc := NewFreecacheClient(config.Cache{})
	assert.True(t, fc.Ping())
}

func TestNewFreecacheClientDefaultsSize(t *testing.T) {
	fc := NewFreecacheClient(config.Cache{FreecacheSizeMB: 0})
	assert.NotNil(t, fc.cache)
}
