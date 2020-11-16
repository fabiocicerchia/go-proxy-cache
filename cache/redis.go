package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/go-redis/redis/v8"
)

type Response struct {
	Method     string
	StatusCode int
	Headers    map[string]interface{}
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

	// test the connection
	_, err := rdb.Ping(ctx).Result()
	return err == nil
}

func Set(key, value string, expiration time.Duration) (bool, error) {
	if rdb == nil {
		return false, fmt.Errorf("Not Connected to Redis")
	}

	err := rdb.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}

func Get(key string) (string, error) {
	if rdb == nil {
		return "", fmt.Errorf("Not Connected to Redis")
	}

	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	return value, nil
}

func LRange(key string) (value []string, err error) {
	if rdb == nil {
		return value, fmt.Errorf("Not Connected to Redis")
	}

	value, err = rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return value, err
	}

	return value, nil
}

func LPush(key string, values []string) error {
	if rdb == nil {
		return fmt.Errorf("Not Connected to Redis")
	}

	return rdb.LPush(ctx, key, values).Err()
}

func Expire(key string, expiration time.Duration) error {
	if rdb == nil {
		return fmt.Errorf("Not Connected to Redis")
	}

	return rdb.Expire(ctx, key, expiration).Err()
}
