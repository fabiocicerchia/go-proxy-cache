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

	return true
}

func StoreFullPage(url string, status int, headers []string, content string, expiration time.Duration) (bool, error) {
	Connect()

	key := url

	err := rdb.Set(ctx, "STATUS@@"+key, status, expiration).Err()
	if err != nil {
		return false, err
	}

	err = rdb.HSet(ctx, "HEADERS@@"+key, headers, expiration).Err()
	if err != nil {
		return false, err
	}

	err = rdb.Set(ctx, "CONTENT@@"+key, content, expiration).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}

func RetrieveFullPage(url string) (status string, headers map[string]string, content string, err error) {
	Connect()

	key := url

	status, err = rdb.Get(ctx, "STATUS@@"+key).Result()
	if err != nil {
		return status, headers, content, err
	}

	headers, err := rdb.HGetAll(ctx, "HEADERS@@"+key).Result()
	if err != nil {
		return status, headers, content, err
	}

	content, err := rdb.Get(ctx, "CONTENT@@"+key).Result()
	if err != nil {
		return status, headers, content, err
	}

	return status, headers, content, nil
}
