package client

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestApplyEvictionPolicyInvalidIsRejected(t *testing.T) {
	logger, hook := test.NewNullLogger()

	rdb := &RedisClient{Name: "testing", logger: logger}

	applyEvictionPolicy(rdb, "not-a-real-policy")

	assert.Equal(t, log.ErrorLevel, hook.LastEntry().Level)
	assert.Contains(t, hook.LastEntry().Message, "Invalid REDIS_EVICTION_POLICY")
}

func TestApplyEvictionPolicyEmptyIsNoop(t *testing.T) {
	logger, hook := test.NewNullLogger()

	rdb := &RedisClient{Name: "testing", logger: logger}

	applyEvictionPolicy(rdb, "")

	assert.Nil(t, hook.LastEntry())
}

func TestValidEvictionPoliciesMatchRedisMaxmemoryPolicy(t *testing.T) {
	for _, policy := range []string{
		"noeviction", "allkeys-lru", "allkeys-lfu", "volatile-lru",
		"volatile-lfu", "allkeys-random", "volatile-random", "volatile-ttl",
	} {
		assert.True(t, validEvictionPolicies[policy], "expected %s to be valid", policy)
	}

	assert.False(t, validEvictionPolicies["bogus"])
}
