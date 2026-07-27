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
	"encoding/json"
	"path"
	"time"

	"github.com/coocood/freecache"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils/base64"
	"github.com/fabiocicerchia/go-proxy-cache/utils/msgpack"
)

const defaultFreecacheSizeMB = 100

// FreecacheClient - In-process, single-instance cache backend. No network
// hop, no external infra, but also no sharing across proxy replicas and no
// wildcard index beyond an O(n) full scan (see DelWildcard below).
type FreecacheClient struct {
	cache *freecache.Cache
}

// NewFreecacheClient - Builds an in-process cache backend sized from config
// (defaults to 100MB when unset).
func NewFreecacheClient(cfg config.Cache) *FreecacheClient {
	sizeMB := cfg.FreecacheSizeMB
	if sizeMB <= 0 {
		sizeMB = defaultFreecacheSizeMB
	}

	return &FreecacheClient{cache: freecache.NewCache(sizeMB * 1024 * 1024)}
}

var _ CacheClient = (*FreecacheClient)(nil)

// Set - Sets a key with a TTL. expiration <= 0 means no expiry.
func (f *FreecacheClient) Set(_ context.Context, key string, value string, expiration time.Duration) (bool, error) {
	err := f.cache.Set([]byte(key), []byte(value), int(expiration.Seconds()))
	return err == nil, err
}

// Get - Gets a key. A cache miss returns ("", nil), matching the Redis
// client's treatment of a missing key as an empty, non-error value.
func (f *FreecacheClient) Get(key string) (string, error) {
	value, err := f.cache.Get([]byte(key))
	if err == freecache.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return string(value), nil
}

// Del - Removes a key.
func (f *FreecacheClient) Del(_ context.Context, key string) error {
	f.cache.Del([]byte(key))
	return nil
}

// DelWildcard - Removes keys matching a glob pattern (the same "*" pattern
// syntax already produced by cache/cache.go for Redis KEYS).
// ponytail: full scan via freecache's iterator, no key index - fine for the
// in-process/small-scale use case this backend targets. Add a prefix index
// if PURGE on a large freecache instance ever shows up as slow.
func (f *FreecacheClient) DelWildcard(_ context.Context, keyPattern string) (int, error) {
	it := f.cache.NewIterator()
	var matched [][]byte
	for entry := it.Next(); entry != nil; entry = it.Next() {
		if ok, _ := path.Match(keyPattern, string(entry.Key)); ok {
			matched = append(matched, entry.Key)
		}
	}

	for _, key := range matched {
		f.cache.Del(key)
	}

	return len(matched), nil
}

// List - Returns a JSON-encoded []string previously stored via Push.
func (f *FreecacheClient) List(key string) ([]string, error) {
	value, err := f.cache.Get([]byte(key))
	if err == freecache.ErrNotFound {
		return []string{}, nil
	}
	if err != nil {
		return []string{}, err
	}

	var values []string
	if err := json.Unmarshal(value, &values); err != nil {
		return []string{}, err
	}

	return values, nil
}

// Push - Appends values to the list stored at key. expiration is not reset
// here to mirror Redis RPUSH, which doesn't touch TTL either; callers set it
// separately via Expire, same as the Redis backend.
func (f *FreecacheClient) Push(_ context.Context, key string, values []string) error {
	existing, err := f.List(key)
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(append(existing, values...))
	if err != nil {
		return err
	}

	return f.cache.Set([]byte(key), encoded, 0)
}

// Expire - Sets a TTL on an existing key.
func (f *FreecacheClient) Expire(key string, expiration time.Duration) error {
	return f.cache.Touch([]byte(key), int(expiration.Seconds()))
}

// Encode - Encodes an object with msgpack, same wire format as the Redis backend.
func (f *FreecacheClient) Encode(obj interface{}) (string, error) {
	value, err := msgpack.Encode(obj)
	if err != nil {
		return "", err
	}

	return base64.Encode(value), nil
}

// Decode - Decodes an object with msgpack, same wire format as the Redis backend.
func (f *FreecacheClient) Decode(encoded string, obj interface{}) error {
	decoded, err := base64.Decode(encoded)
	if err != nil {
		return err
	}

	return msgpack.Decode(decoded, obj)
}

// Ping - Always available; there's no network connection to check.
func (f *FreecacheClient) Ping() bool {
	return true
}
