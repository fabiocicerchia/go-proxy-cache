package engine

import (
	"context"
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
		err := rdb.Ping(ctx).Err()

		if err != nil && err.Error() != "redis: client is closed" {
			return false
		}
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     config.Host + ":" + config.Port,
		Password: config.Password,
		DB:       config.DB,
	})

	// test the connection
	return Ping()
}

// Close - Closes the connection.
func Close() error {
	return rdb.Close()
}

// PurgeAll - Purges all the existing keys on a DB.
func PurgeAll() (bool, error) {
	_, err := config.CB().Execute(func() (interface{}, error) {
		err := rdb.FlushDB(ctx).Err()
		return nil, err
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

// Ping - Tests the connection.
func Ping() bool {
	_, err := config.CB().Execute(func() (interface{}, error) {
		err := rdb.Ping(ctx).Err()
		return nil, err
	})

	return err == nil
}

// Set - Sets a key, with certain value, with TTL for expiring.
func Set(key string, value string, expiration time.Duration) (bool, error) {
	_, err := config.CB().Execute(func() (interface{}, error) {
		err := rdb.Set(ctx, key, value, expiration).Err()
		return nil, err
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

// Get - Gets a key.
func Get(key string) (string, error) {
	value, err := config.CB().Execute(func() (interface{}, error) {
		value, err := rdb.Get(ctx, key).Result()
		return value, err
	})

	if err != nil {
		return "", err
	}

	return value.(string), nil
}

// Del - Removes a key.
func Del(key string) error {
	_, err := config.CB().Execute(func() (interface{}, error) {
		err := rdb.Del(ctx, key).Err()
		return nil, err
	})

	return err
}

// DelWildcard - Removes the matching keys based on a pattern.
func DelWildcard(key string) (int, error) {
	k, err := config.CB().Execute(func() (interface{}, error) {
		keys, err := rdb.Keys(ctx, key).Result()
		return keys, err
	})

	if err != nil {
		return 0, err
	}

	keys := k.([]string)
	l := len(keys)

	if l == 0 {
		return l, nil
	}

	_, errDel := config.CB().Execute(func() (interface{}, error) {
		err := rdb.Del(ctx, keys...).Err()
		return nil, err
	})

	return l, errDel
}

// List - Returns the values in a list.
func List(key string) ([]string, error) {
	value, err := config.CB().Execute(func() (interface{}, error) {
		value, err := rdb.LRange(ctx, key, 0, -1).Result()
		return value, err
	})

	if err != nil {
		return []string{}, err
	}

	return value.([]string), nil
}

// Push - Append values to a list.
func Push(key string, values []string) error {
	_, err := config.CB().Execute(func() (interface{}, error) {
		err := rdb.RPush(ctx, key, values).Err()
		return nil, err
	})

	return err
}

// Expire - Sets a TTL on a key.
func Expire(key string, expiration time.Duration) error {
	_, err := config.CB().Execute(func() (interface{}, error) {
		err := rdb.Expire(ctx, key, expiration).Err()
		return nil, err
	})

	return err
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
