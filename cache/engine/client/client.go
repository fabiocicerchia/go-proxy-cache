package client

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"fmt"
	"time"

	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils/base64"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
	"github.com/fabiocicerchia/go-proxy-cache/utils/msgpack"
)

var ctx = context.Background()

// RedisClient - Redis Client structure.
type RedisClient struct {
	*goredislib.Client
	*redsync.Redsync
	Name  string
	Mutex map[string]*redsync.Mutex
}

// Connect - Connects to DB.
func Connect(connName string, config config.Cache) *RedisClient {
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
	}

	return rdb
}

// Close - Closes the connection.
func (rdb *RedisClient) Close() error {
	return rdb.Client.Close()
}

func (rdb *RedisClient) getMutex(key string) *redsync.Mutex {
	mutexname := fmt.Sprintf("mutex-%s", key)
	if _, ok := rdb.Mutex[mutexname]; !ok {
		rdb.Mutex[mutexname] = rdb.Redsync.NewMutex(mutexname)
	}

	return rdb.Mutex[mutexname]
}

func (rdb *RedisClient) lock(key string) error {
	if err := rdb.getMutex(key).Lock(); err != nil {
		log.Errorf("Lock Error on %s: %s", key, err)
		return err
	}

	return nil
}

func (rdb *RedisClient) unlock(key string) error {
	if ok, err := rdb.getMutex(key).Unlock(); !ok || err != nil {
		log.Errorf("Unlock Error on %s: %s", key, err)
		return err
	}

	return nil
}

// PurgeAll - Purges all the existing keys on a DB.
func (rdb *RedisClient) PurgeAll() (bool, error) {
	_, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		err := rdb.Client.FlushDB(ctx).Err()
		return nil, err
	})

	return err == nil, err
}

// Ping - Tests the connection.
func (rdb *RedisClient) Ping() bool {
	_, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		err := rdb.Client.Ping(ctx).Err()
		return nil, err
	})

	return err == nil
}

// Set - Sets a key, with certain value, with TTL for expiring (soft and hard eviction).
func (rdb *RedisClient) Set(key string, value string, expiration time.Duration) (bool, error) {
	_, err := circuitbreaker.CB(rdb.Name).Execute(rdb.doSet(key, value, expiration))

	return err == nil, err
}

func (rdb *RedisClient) doSet(key string, value string, expiration time.Duration) func() (interface{}, error) {
	return func() (interface{}, error) {
		if errLock := rdb.lock(key); errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.Set(ctx, key, value, expiration).Err()

		if errUnlock := rdb.unlock(key); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	}
}

// Get - Gets a key.
func (rdb *RedisClient) Get(key string) (string, error) {
	value, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		value, err := rdb.Client.Get(ctx, key).Result()
		if value == "" && err != nil && err.Error() == "redis: nil" {
			return value, nil
		}

		return value, err
	})

	return value.(string), err
}

// Del - Removes a key.
func (rdb *RedisClient) Del(key string) error {
	_, err := rdb.deleteKeys(key, []string{key})

	return err
}

// DelWildcard - Removes the matching keys based on a pattern.
func (rdb *RedisClient) DelWildcard(key string) (int, error) {
	k, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		keys, err := rdb.Client.Keys(ctx, key).Result()
		return keys, err
	})

	if err != nil {
		return 0, nil
	}

	return rdb.deleteKeys(key, k.([]string))
}

// DelWildcard - Removes the matching keys based on a pattern.
func (rdb *RedisClient) deleteKeys(keyID string, keys []string) (int, error) {
	l := len(keys)

	if l == 0 {
		return 0, nil
	}

	_, errDel := circuitbreaker.CB(rdb.Name).Execute(rdb.doDeleteKeys(keyID, keys))

	return l, errDel
}

func (rdb *RedisClient) doDeleteKeys(keyID string, keys []string) func() (interface{}, error) {
	return func() (interface{}, error) {
		if errLock := rdb.lock(keyID); errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.Del(ctx, keys...).Err()

		if errUnlock := rdb.unlock(keyID); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	}
}

// List - Returns the values in a list.
func (rdb *RedisClient) List(key string) ([]string, error) {
	value, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		value, err := rdb.Client.LRange(ctx, key, 0, -1).Result()
		return value, err
	})

	if err != nil {
		return []string{}, err
	}

	return value.([]string), nil
}

// Push - Append values to a list.
func (rdb *RedisClient) Push(key string, values []string) error {
	_, err := circuitbreaker.CB(rdb.Name).Execute(rdb.doPushKey(key, values))

	return err
}

func (rdb *RedisClient) doPushKey(key string, values []string) func() (interface{}, error) {
	return func() (interface{}, error) {
		if errLock := rdb.lock(key); errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.RPush(ctx, key, values).Err()

		if errUnlock := rdb.unlock(key); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	}
}

// Expire - Sets a TTL on a key (hard eviction only).
func (rdb *RedisClient) Expire(key string, expiration time.Duration) error {
	_, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
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
