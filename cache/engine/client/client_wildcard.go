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

	goredislib "github.com/go-redis/redis/v8"

	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

// DelWildcard - Removes the matching keys based on a pattern.
func (rdb *RedisClient) DelWildcard(ctx context.Context, key string) (int, error) {
	if rdb.Client.ClusterSlots(ctx).Err() == nil {
		return rdb.deleteClusterKeys(ctx, key)
	}
		k, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(func() (interface{}, error) {
			keys, err := rdb.Client.Keys(ctx, key).Result()
			return keys, err
		})

		if err != nil {
			return 0, nil
		}

		return rdb.deleteKeys(ctx, key, k.([]string))
}

func (rdb *RedisClient) deleteClusterKeys(ctx context.Context, key string) (int, error) {
	clusterClient := rdb.Client.(*goredislib.ClusterClient)
	deletedKeys := Counter{}

	err := clusterClient.ForEachShard(ctx, func(ctx context.Context, client *goredislib.Client) error {
		_, err := circuitbreaker.CB(rdb.Name, rdb.logger).Execute(func() (interface{}, error) {
			keys, err := client.Keys(ctx, key).Result()
			if err != nil {
				rdb.logger.Errorf("Error removing keys with pattern: %s on node: %s", key, client)
			}
			deletedKeysByNode, err := rdb.deleteKeysByShard(ctx, key, keys, client)
			if err == nil {
				deletedKeys.increment(deletedKeysByNode)
			}
			return nil, err
		})
		return err
	})

	if err != nil {
		return 0, nil
	}

	return deletedKeys.counter, nil
}

func (rdb *RedisClient) deleteKeysByShard(ctx context.Context, key string, keys []string, client *goredislib.Client) (int, error) {
	if (len(keys) == 0 && keys != nil) {
		rdb.logger.Printf("Keys with pattern: %s not found in node: %s\n", key, client)
	}
	deletedKeysByNode, err := rdb.deleteKeys(ctx, key, keys)
	if len(keys) > 0 {
		rdb.logger.Printf("Keys with pattern: %s removed from node: %v\n", key, client)
	}
	return deletedKeysByNode, err
}

// DelWildcard - Removes the matching keys based on a pattern.
func (rdb *RedisClient) deleteKeys(ctx context.Context, keyID string, keys []string) (int, error) {
	l := len(keys)
	var errDel error

	if l == 0 {
		return 0, nil
	}

	if rdb.Client.ClusterSlots(ctx).Err() != nil {
		_, errDel = circuitbreaker.CB(rdb.Name, rdb.logger).Execute(rdb.doDeleteKeys(ctx, keyID, keys))
	} else {
		_, errDel = circuitbreaker.CB(rdb.Name, rdb.logger).Execute(rdb.doDeleteNodeKeys(ctx, keys))
	}

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

func (rdb *RedisClient) doDeleteNodeKeys(ctx context.Context, keys []string) func() (interface{}, error) {
	return func() (interface{}, error) {
		var removingError error

		for _, currentKey := range keys {
			if errLock := rdb.lock(ctx, currentKey); errLock != nil {
				return nil, errLock
			}
			removingError = rdb.Client.Del(ctx, currentKey).Err()
			if errUnlock := rdb.unlock(ctx, currentKey); errUnlock != nil {
				return nil, errUnlock
			}
		}

		return nil, removingError
	}
}
