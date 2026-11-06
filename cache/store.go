package cache

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

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
)

// Store - The storage the caching rules depend on; the Redis client satisfies it.
type Store interface {
	Encode(obj interface{}) (string, error)
	Decode(encoded string, obj interface{}) error
	Set(ctx context.Context, key string, value string, expiration time.Duration) (bool, error)
	Get(key string) (string, error)
	Del(ctx context.Context, key string) error
	DelWildcard(ctx context.Context, key string) (int, error)
	List(key string) ([]string, error)
	Push(ctx context.Context, key string, values []string) error
	Expire(key string, expiration time.Duration) error
}

// GetStore - Returns the Store for a domain, or nil when there is none.
// Defaults to the Redis engine; replace it to run the rules without Redis.
var GetStore = func(domainID string) Store {
	// A nil *RedisClient in a Store would not compare equal to nil.
	if conn := engine.GetConn(domainID); conn != nil {
		return conn
	}

	return nil
}
