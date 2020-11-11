package cache_redis

import (
	"context"
	"strconv"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

var rdb *redis.Client

func Connect() bool {
	if rdb != nil {
		return true
	}

	host := utils.GetEnv("REDIS_HOST", "")
	port := utils.GetEnv("REDIS_PORT", "6379")
	password := utils.GetEnv("REDIS_PASSWORD", "")

	db, err := strconv.Atoi(utils.GetEnv("REDIS_DB", "0"))
	if err != nil {
		db = 0
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       db,
	})
}

func StoreFullPage(url string, content string, expiration time.Duration) (bool, error) {
	Connect()

	key := url

	err := rdb.Set(ctx, key, content, expiration).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}

func RetrieveFullPage(url string) (string, error) {
	Connect()

	key := url

	val, err := rdb.Get(ctx, key).Result()
	return val, err
}
