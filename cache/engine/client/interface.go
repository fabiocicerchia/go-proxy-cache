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
	"context"
	"time"
)

// CacheClient - Minimal set of operations a cache backend must support.
// Method set matches exactly what cache/cache.go and the health check use
// against *RedisClient today - nothing speculative.
type CacheClient interface {
	Set(ctx context.Context, key string, value string, expiration time.Duration) (bool, error)
	Get(key string) (string, error)
	Del(ctx context.Context, key string) error
	DelWildcard(ctx context.Context, key string) (int, error)
	List(key string) ([]string, error)
	Push(ctx context.Context, key string, values []string) error
	Expire(key string, expiration time.Duration) error
	Encode(obj interface{}) (string, error)
	Decode(encoded string, obj interface{}) error
	Ping() bool
}

var _ CacheClient = (*RedisClient)(nil)
