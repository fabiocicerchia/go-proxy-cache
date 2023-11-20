//go:build all || functional
// +build all functional

package client_test

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
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine/client"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

// this is to verify any possible data race condition
const clusterRedisConnName = "testing"
const clusterClashingKey = "test"

func initLogsCluster() {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
}

func TestClusterCircuitBreakerWithPingTimeout(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	assert.Equal(t, "closed", circuit_breaker.CB(clusterRedisConnName, logger.GetGlobal()).State().String())

	val := rdb.Ping()
	assert.True(t, val)
	assert.Equal(t, "closed", circuit_breaker.CB(clusterRedisConnName, logger.GetGlobal()).State().String())

	_ = rdb.Close()

	val = rdb.Ping()
	assert.False(t, val)
	assert.Equal(t, "half-open", circuit_breaker.CB(clusterRedisConnName, logger.GetGlobal()).State().String())

	rdb = client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	val = rdb.Ping()
	assert.True(t, val)
	assert.Equal(t, "closed", circuit_breaker.CB(clusterRedisConnName, logger.GetGlobal()).State().String())
}

func TestCloseCluster(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	assert.True(t, rdb.Ping())

	rdb.Close()

	assert.False(t, rdb.Ping())
}

func TestClusterSetGet(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	done, err := rdb.Set(context.Background(), clusterClashingKey, "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := rdb.Get(clusterClashingKey)
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
}

func TestClusterSetGetWithExpiration(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	done, err := rdb.Set(context.Background(), clusterClashingKey, "sample", 1*time.Millisecond)
	assert.True(t, done)
	assert.Nil(t, err)

	time.Sleep(10 * time.Millisecond) // let it expire in Redis

	value, err := rdb.Get(clusterClashingKey)
	assert.Equal(t, "", value)
	assert.Nil(t, err)
}

func TestClusterDel(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	done, err := rdb.Set(context.Background(), clusterClashingKey, "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := rdb.Get(clusterClashingKey)
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)

	err = rdb.Del(context.Background(), clusterClashingKey)
	assert.Nil(t, err)

	value, err = rdb.Get(clusterClashingKey)
	assert.Equal(t, "", value)
	assert.Nil(t, err)
}

func TestClusterExpire(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	done, err := rdb.Set(context.Background(), clusterClashingKey, "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	// redis: commands.go:36: specified duration is 100ms, but minimal supported value is 1s - truncating to 1s
	err = rdb.Expire(clusterClashingKey, 100*time.Millisecond)
	assert.Nil(t, err)

	time.Sleep(2 * time.Second)

	value, err := rdb.Get(clusterClashingKey)
	assert.Equal(t, "", value)
	assert.Nil(t, err)
}

func TestClusterPushList(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	err := rdb.Push(context.Background(), clusterClashingKey, []string{"a", "b", "c"})
	assert.Nil(t, err)

	value, err := rdb.List(clusterClashingKey)
	assert.Equal(t, []string{"a", "b", "c"}, value)
	assert.Nil(t, err)
}

func TestClusterDelWildcardNoMatch(t *testing.T) {
    initLogsCluster()

    cfg := config.Configuration{
        Cache: config.Cache{
            Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
            DB:    0,
        },
        CircuitBreaker: circuit_breaker.CircuitBreaker{
            Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
            FailureRate: 0.5,              // 1 out of 2 fails, or more
            Interval:    0,                // doesn't clears counts
            Timeout:     time.Duration(1), // clears state immediately
        },
    }

    circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

    rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

    done, err := rdb.Set(context.Background(), "test_1", "sample", 0)
    assert.True(t, done)
    assert.Nil(t, err)
    done, err = rdb.Set(context.Background(), "test_2", "sample", 0)
    assert.True(t, done)
    assert.Nil(t, err)
    done, err = rdb.Set(context.Background(), "test_3", "sample", 0)
    assert.True(t, done)
    assert.Nil(t, err)

    value, err := rdb.Get("test_1")
    assert.Equal(t, "sample", value)
    assert.Nil(t, err)
    value, err = rdb.Get("test_2")
    assert.Equal(t, "sample", value)
    assert.Nil(t, err)
    value, err = rdb.Get("test_3")
    assert.Equal(t, "sample", value)
    assert.Nil(t, err)

    len, err := rdb.DelWildcard(context.Background(), "missing_*")
    assert.Equal(t, 0, len)
    assert.Nil(t, err)

    value, err = rdb.Get("test_1")
    assert.Equal(t, "sample", value)
    assert.Nil(t, err)
    value, err = rdb.Get("test_2")
    assert.Equal(t, "sample", value)
    assert.Nil(t, err)
    value, err = rdb.Get("test_3")
    assert.Equal(t, "sample", value)
    assert.Nil(t, err)
}

func TestClusterDelWildcard(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	done, err := rdb.Set(context.Background(), "test_1", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set(context.Background(), "test_2", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set(context.Background(), "test_3", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := rdb.Get("test_1")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
	value, err = rdb.Get("test_2")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
	value, err = rdb.Get("test_3")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)

	len, err := rdb.DelWildcard(context.Background(), "test_*")
	assert.Equal(t, 3, len)
	assert.Nil(t, err)

	value, err = rdb.Get("test_1")
	assert.Equal(t, "", value)
	assert.Nil(t, err)
	value, err = rdb.Get("test_2")
	assert.Equal(t, "", value)
	assert.Nil(t, err)
	value, err = rdb.Get("test_3")
	assert.Equal(t, "", value)
	assert.Nil(t, err)
}

func TestClusterPurgeAll(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	done, err := rdb.Set(context.Background(), "test_1", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set(context.Background(), "test_2", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set(context.Background(), "test_3", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := rdb.Get("test_1")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
	value, err = rdb.Get("test_2")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
	value, err = rdb.Get("test_3")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)

	done, err = rdb.PurgeAll()
	assert.True(t, done)
	assert.Nil(t, err)

	value, err = rdb.Get("test_1")
	assert.Equal(t, "", value)
	assert.Nil(t, err)
	value, err = rdb.Get("test_2")
	assert.Equal(t, "", value)
	assert.Nil(t, err)
	value, err = rdb.Get("test_3")
	assert.Equal(t, "", value)
	assert.Nil(t, err)
}

func TestClusterEncodeDecode(t *testing.T) {
	initLogsCluster()

	cfg := config.Configuration{
		Cache: config.Cache{
			Hosts: strings.Split(utils.GetEnv("REDIS_HOSTS", "172.20.0.36:6379,172.20.0.37:6379,172.20.0.38:6379"), ","),
			DB:    0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker(clusterRedisConnName, cfg.CircuitBreaker, logger.GetGlobal())

	rdb := client.Connect(clusterRedisConnName, cfg.Cache, log.StandardLogger())

	str := []byte("test string")
	var decoded []byte

	encoded, err := rdb.Encode(str)
	assert.Nil(t, err)
	err = rdb.Decode(encoded, &decoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}
