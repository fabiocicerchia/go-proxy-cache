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
	"fmt"
	"strings"
	"time"

	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	goredis "github.com/go-redsync/redsync/v4/redis/goredis/v8"
	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/utils/base64"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
	"github.com/fabiocicerchia/go-proxy-cache/utils/msgpack"
)

var ctx = context.Background()

// RedisClient - Redis Client structure.
type RedisClient struct {
	*goredislib.Client
	*redsync.Redsync
	Name   string
	Mutex  map[string]*redsync.Mutex
	logger *log.Logger
}

// Connect - Connects to DB.
func Connect(connName string, config config.Cache, logger *log.Logger) *RedisClient {
	client := goredislib.NewClient(&goredislib.Options{
		Addr:     config.Host + ":" + config.Port,
		Password: config.Password,
		DB:       config.DB,
	})
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)

	rdb := &RedisClient{
		Name:    connName,
		Client:  client,
		Redsync: rs,
		Mutex:   make(map[string]*redsync.Mutex),
		logger:  logger,
	}

	return rdb
}

// Close - Closes the connection.
func (rdb *RedisClient) Close() error {
	return rdb.Client.Close()
}

func (rdb *RedisClient) getMutex(key string) *redsync.Mutex {
	mutexname := fmt.Sprintf("mutex-%s-%s", rdb.Name, key)
	if _, ok := rdb.Mutex[mutexname]; !ok {
		rdb.Mutex[mutexname] = rdb.Redsync.NewMutex(mutexname)
	}

	return rdb.Mutex[mutexname]
}

func (rdb *RedisClient) lock(ctx context.Context, key string) error {
	if err := rdb.getMutex(key).Lock(); err != nil {
		escapedKey := strings.Replace(key, "\n", "", -1)
		escapedKey = strings.Replace(escapedKey, "\r", "", -1)
		rdb.logger.Errorf("Lock Error on %s: %s", escapedKey, err)
		telemetry.From(ctx).RegisterEventWithData("Lock Error", map[string]string{
			"key":   key,
			"error": err.Error(),
		})
		return err
	}

	return nil
}

func (rdb *RedisClient) unlock(ctx context.Context, key string) error {
	if ok, err := rdb.getMutex(key).Unlock(); !ok || err != nil {
		escapedKey := strings.Replace(key, "\n", "", -1)
		escapedKey = strings.Replace(escapedKey, "\r", "", -1)
		rdb.logger.Errorf("Unlock Error on %s: %s", escapedKey, err)
		telemetry.From(ctx).RegisterEventWithData("Lock Error", map[string]string{
			"key":   key,
			"error": err.Error(),
		})
		return err
	}

	return nil
}

// PurgeAll - Purges all the existing keys on a DB.
func (rdb *RedisClient) PurgeAll() (bool, error) {
	_, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(func() (interface{}, error) {
		err := rdb.Client.FlushDB(ctx).Err()
		return nil, err
	})

	return err == nil, err
}

// Ping - Tests the connection.
func (rdb *RedisClient) Ping() bool {
	_, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(func() (interface{}, error) {
		err := rdb.Client.Ping(ctx).Err()
		return nil, err
	})

	return err == nil
}

// Set - Sets a key, with certain value, with TTL for expiring (soft and hard eviction).
func (rdb *RedisClient) Set(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	_, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(rdb.doSet(ctx, key, value, expiration))

	return err == nil, err
}

func (rdb *RedisClient) doSet(ctx context.Context, key string, value string, expiration time.Duration) func() (interface{}, error) {
	return func() (interface{}, error) {
		if errLock := rdb.lock(ctx, key); errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.Set(ctx, key, value, expiration).Err()

		if errUnlock := rdb.unlock(ctx, key); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	}
}

// Get - Gets a key.
func (rdb *RedisClient) Get(key string) (string, error) {
	value, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(func() (interface{}, error) {
		value, err := rdb.Client.Get(ctx, key).Result()
		if value == "" && err != nil && err.Error() == "redis: nil" {
			return value, nil
		}

		return value, err
	})

	return value.(string), err
}

// Del - Removes a key.
func (rdb *RedisClient) Del(ctx context.Context, key string) error {
	_, err := rdb.deleteKeys(ctx, key, []string{key})

	return err
}

// DelWildcard - Removes the matching keys based on a pattern.
func (rdb *RedisClient) DelWildcard(ctx context.Context, key string) (int, error) {
	k, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(func() (interface{}, error) {
		keys, err := rdb.Client.Keys(ctx, key).Result()
		return keys, err
	})

	if err != nil {
		return 0, nil
	}

	return rdb.deleteKeys(ctx, key, k.([]string))
}

// DelWildcard - Removes the matching keys based on a pattern.
func (rdb *RedisClient) deleteKeys(ctx context.Context, keyID string, keys []string) (int, error) {
	l := len(keys)

	if l == 0 {
		return 0, nil
	}

	_, errDel := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(rdb.doDeleteKeys(ctx, keyID, keys))

	return l, errDel
}

func (rdb *RedisClient) doDeleteKeys(ctx context.Context, keyID string, keys []string) func() (interface{}, error) {
	return func() (interface{}, error) {
		if errLock := rdb.lock(ctx, keyID); errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.Del(ctx, keys...).Err()

		if errUnlock := rdb.unlock(ctx, keyID); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	}
}

// List - Returns the values in a list.
func (rdb *RedisClient) List(key string) ([]string, error) {
	value, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(func() (interface{}, error) {
		value, err := rdb.Client.LRange(ctx, key, 0, -1).Result()
		return value, err
	})

	if err != nil {
		return []string{}, err
	}

	return value.([]string), nil
}

// Push - Append values to a list.
func (rdb *RedisClient) Push(ctx context.Context, key string, values []string) error {
	_, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(rdb.doPushKey(ctx, key, values))

	return err
}

func (rdb *RedisClient) doPushKey(ctx context.Context, key string, values []string) func() (interface{}, error) {
	return func() (interface{}, error) {
		if errLock := rdb.lock(ctx, key); errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.RPush(ctx, key, values).Err()

		if errUnlock := rdb.unlock(ctx, key); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	}
}

// Expire - Sets a TTL on a key (hard eviction only).
func (rdb *RedisClient) Expire(key string, expiration time.Duration) error {
	_, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(func() (interface{}, error) {
		err := rdb.Client.Expire(ctx, key, expiration).Err()
		return nil, err
	})

	return err
}

// Encode - Encodes an object with msgpack.
func (rdb *RedisClient) Encode(obj interface{}) (string, error) {
	value, err := msgpack.Encode(obj)
	if err != nil {
		return "", err
	}

	encoded := base64.Encode(value)

	return encoded, nil
}

// Decode - Decodes an object with msgpack.
func (rdb *RedisClient) Decode(encoded string, obj interface{}) error {
	decoded, err := base64.Decode(encoded)
	if err != nil {
		return err
	}

	err = msgpack.Decode(decoded, obj)

	return err
}
