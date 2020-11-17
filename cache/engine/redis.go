package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var rdb *redis.Client

// Connect - Connects to DB.
func Connect(config config.Cache) bool {
	if rdb != nil {
		return Ping()
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

// Close - Closes the connection.
func Close() error {
	return rdb.Close()
}

// PurgeAll - Purges all the existing keys on a DB.
func PurgeAll() (bool, error) {
	if rdb == nil {
		return false, fmt.Errorf("Not Connected to Redis")
	}

	err := rdb.FlushDB(ctx).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}

// Ping - Tests the connection.
func Ping() bool {
	if rdb == nil {
		return false
	}

	_, err := rdb.Ping(ctx).Result()
	return err == nil
}

// Set - Sets a key, with certain value, with TTL for expiring.
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

// Get - Gets a key.
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

// Del - Removes a key.
func Del(key string) error {
	if rdb == nil {
		return fmt.Errorf("Not Connected to Redis")
	}

	return rdb.Del(ctx, key).Err()
}

// DelWildcard - Removes the matching keys based on a pattern.
func DelWildcard(key string) error {
	if rdb == nil {
		return fmt.Errorf("Not Connected to Redis")
	}

	keys, err := rdb.Keys(ctx, key).Result()
	if err != nil {
		return err
	}

	return rdb.Del(ctx, keys...).Err()
}

// List - Returns the values in a list.
func List(key string) (value []string, err error) {
	if rdb == nil {
		return value, fmt.Errorf("Not Connected to Redis")
	}

	value, err = rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return value, err
	}

	return value, nil
}

// Push - Append values to a list.
func Push(key string, values []string) error {
	if rdb == nil {
		return fmt.Errorf("Not Connected to Redis")
	}

	return rdb.LPush(ctx, key, values).Err()
}

// Expire - Sets a TTL on a key.
func Expire(key string, expiration time.Duration) error {
	if rdb == nil {
		return fmt.Errorf("Not Connected to Redis")
	}

	return rdb.Expire(ctx, key, expiration).Err()
}

// Encode - Encodes an object with msgpack.
func Encode(obj interface{}) (string, error) {
	value, err := utils.MsgpackEncode(obj)
	if err != nil {
		return "", err
	}

	encoded := utils.Base64Encode(value)

	return encoded, nil
}

// Decode - Decodes an object with msgpack.
func Decode(encoded string, obj interface{}) error {
	decoded, err := utils.Base64Decode(encoded)
	if err != nil {
		return err
	}

	err = utils.MsgpackDecode(decoded, obj)
	return err
}
