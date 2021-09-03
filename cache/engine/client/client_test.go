// +build all functional

package client_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine/client"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuit_breaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
	"github.com/stretchr/testify/assert"
)

func initLogs() {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
}

func TestCircuitBreakerWithPingTimeout(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	assert.Equal(t, "closed", circuit_breaker.CB("testing").State().String())

	val := rdb.Ping()
	assert.True(t, val)
	assert.Equal(t, "closed", circuit_breaker.CB("testing").State().String())

	_ = rdb.Close()

	val = rdb.Ping()
	assert.False(t, val)
	assert.Equal(t, "half-open", circuit_breaker.CB("testing").State().String())

	rdb = client.Connect("testing", cfg.Cache, log.StandardLogger())

	val = rdb.Ping()
	assert.True(t, val)
	assert.Equal(t, "closed", circuit_breaker.CB("testing").State().String())
}

func TestClose(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	assert.True(t, rdb.Ping())

	rdb.Close()

	assert.False(t, rdb.Ping())
}

func TestSetGet(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	done, err := rdb.Set("test", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := rdb.Get("test")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
}

func TestSetGetWithExpiration(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	done, err := rdb.Set("test", "sample", 1*time.Millisecond)
	assert.True(t, done)
	assert.Nil(t, err)

	time.Sleep(10 * time.Millisecond)

	value, err := rdb.Get("test")
	assert.Equal(t, "", value)
	assert.Nil(t, err)
}

func TestDel(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	done, err := rdb.Set("test", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := rdb.Get("test")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)

	err = rdb.Del("test")
	assert.Nil(t, err)

	value, err = rdb.Get("test")
	assert.Equal(t, "", value)
	assert.Nil(t, err)
}

func TestExpire(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	done, err := rdb.Set("test", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	err = rdb.Expire("test", 1*time.Second)
	assert.Nil(t, err)

	time.Sleep(1500 * time.Millisecond)

	value, err := rdb.Get("test")
	assert.Equal(t, "", value)
	assert.Nil(t, err)
}

func TestPushList(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	err := rdb.Push("test", []string{"a", "b", "c"})
	assert.Nil(t, err)

	value, err := rdb.List("test")
	assert.Equal(t, []string{"a", "b", "c"}, value)
	assert.Nil(t, err)
}

func TestDelWildcardNoMatch(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	done, err := rdb.Set("test_1", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set("test_2", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set("test_3", "sample", 0)
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

	len, err := rdb.DelWildcard("missing_*")
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

func TestDelWildcard(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	done, err := rdb.Set("test_1", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set("test_2", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set("test_3", "sample", 0)
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

	len, err := rdb.DelWildcard("test_*")
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

func TestPurgeAll(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	done, err := rdb.Set("test_1", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set("test_2", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = rdb.Set("test_3", "sample", 0)
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

func TestEncodeDecode(t *testing.T) {
	initLogs()

	cfg := config.Configuration{
		Cache: config.Cache{
			Host: utils.GetEnv("REDIS_HOST", "localhost"),
			Port: "6379",
			DB:   0,
		},
		CircuitBreaker: circuit_breaker.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	circuit_breaker.InitCircuitBreaker("testing", cfg.CircuitBreaker)

	rdb := client.Connect("testing", cfg.Cache, log.StandardLogger())

	str := []byte("test string")
	var decoded []byte

	encoded, err := rdb.Encode(str)
	assert.Nil(t, err)
	err = rdb.Decode(encoded, &decoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}
