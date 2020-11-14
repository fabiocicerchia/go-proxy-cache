package cache_redis

import (
	"context"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/go-redis/redis/v8"
)

type Response struct {
	StatusCode int
	Headers    map[string]string
	Content    string
}

var ctx = context.Background()

var rdb *redis.Client

func Connect(config config.Cache) bool {
	if rdb != nil {
		// test the connection
		_, err := rdb.Ping(ctx).Result()
		return err == nil
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     config.Host + ":" + config.Port,
		Password: config.Password,
		DB:       config.DB,
	})

	return true
}

func Set(key, value string, expiration time.Duration) (bool, error) {
	err := rdb.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}

func Get(key string) (string, error) {
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	return value, nil
}
