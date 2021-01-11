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

// RedisClient - Redis Client structure
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

func (rdb *RedisClient) lock(key string) (*redsync.Mutex, error) {
	mutex := rdb.getMutex(key)

	if err := mutex.Lock(); err != nil {
		log.Errorf("Lock Error on %s: %s", key, err)
		return nil, err
	}

	return mutex, nil
}

func (rdb *RedisClient) unlock(mutex *redsync.Mutex, key string) error {
	if ok, err := mutex.Unlock(); !ok || err != nil {
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

// Set - Sets a key, with certain value, with TTL for expiring.
func (rdb *RedisClient) Set(key string, value string, expiration time.Duration) (bool, error) {
	_, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		mutex, errLock := rdb.lock(key)
		if errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.Set(ctx, key, value, expiration).Err()

		if errUnlock := rdb.unlock(mutex, key); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	})

	return err == nil, err
}

// Get - Gets a key.
func (rdb *RedisClient) Get(key string) (string, error) {
	value, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		value, err := rdb.Client.Get(ctx, key).Result()
		if value == "" && err != nil && err.Error() == "redis: nil" {
			return "", nil
		}

		return value, err
	})

	if err != nil {
		return "", err
	}

	return value.(string), nil
}

// Del - Removes a key.
func (rdb *RedisClient) Del(key string) error {
	_, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		mutex, errLock := rdb.lock(key)
		if errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.Del(ctx, key).Err()

		if errUnlock := rdb.unlock(mutex, key); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	})

	return err
}

// DelWildcard - Removes the matching keys based on a pattern.
func (rdb *RedisClient) DelWildcard(key string) (int, error) {
	k, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		keys, err := rdb.Client.Keys(ctx, key).Result()
		return keys, err
	})

	keys := k.([]string)
	l := len(keys)

	if err != nil || l == 0 {
		return l, nil
	}

	_, errDel := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		mutex, errLock := rdb.lock(key)
		if errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.Del(ctx, keys...).Err()

		if errUnlock := rdb.unlock(mutex, key); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	})

	return l, errDel
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
	_, err := circuitbreaker.CB(rdb.Name).Execute(func() (interface{}, error) {
		mutex, errLock := rdb.lock(key)
		if errLock != nil {
			return nil, errLock
		}

		err := rdb.Client.RPush(ctx, key, values).Err()

		if errUnlock := rdb.unlock(mutex, key); errUnlock != nil {
			return nil, errUnlock
		}

		return nil, err
	})

	return err
}

// Expire - Sets a TTL on a key.
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
