// +build functional

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

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine/client"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerWithPingTimeout(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

	assert.Equal(t, "closed", config.CB().State().String())

	val := rdb.Ping()
	assert.True(t, val)
	assert.Equal(t, "closed", config.CB().State().String())

	_ = rdb.Close()

	val = rdb.Ping()
	assert.False(t, val)
	assert.Equal(t, "half-open", config.CB().State().String())

	rdb = client.Connect(config.Config.Cache)

	val = rdb.Ping()
	assert.True(t, val)
	assert.Equal(t, "closed", config.CB().State().String())
}

func TestClose(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

	assert.True(t, rdb.Ping())

	rdb.Close()

	assert.False(t, rdb.Ping())
}

func TestSetGet(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

	done, err := rdb.Set("test", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := rdb.Get("test")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
}

func TestSetGetWithExpiration(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

	done, err := rdb.Set("test", "sample", 1*time.Millisecond)
	assert.True(t, done)
	assert.Nil(t, err)

	time.Sleep(10 * time.Millisecond)

	value, err := rdb.Get("test")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
}

func TestDel(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

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
	assert.Equal(t, "redis: nil", err.Error())
}

func TestExpire(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

	done, err := rdb.Set("test", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	err = rdb.Expire("test", 1*time.Second)
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)

	value, err := rdb.Get("test")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
}

func TestPushList(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

	err := rdb.Push("test", []string{"a", "b", "c"})
	assert.Nil(t, err)

	value, err := rdb.List("test")
	assert.Equal(t, []string{"a", "b", "c"}, value)
	assert.Nil(t, err)
}

func TestDelWildcard(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

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
	assert.Equal(t, "redis: nil", err.Error())
	value, err = rdb.Get("test_2")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
	value, err = rdb.Get("test_3")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
}

func TestPurgeAll(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

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
	assert.Equal(t, "redis: nil", err.Error())
	value, err = rdb.Get("test_2")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
	value, err = rdb.Get("test_3")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
}

func TestEncodeDecode(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		CircuitBreaker: config.CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     time.Duration(1), // clears state immediately
		},
	}

	config.InitCircuitBreaker(config.Config.CircuitBreaker)

	rdb := client.Connect(config.Config.Cache)

	str := []byte("test string")
	var decoded []byte

	encoded, err := rdb.Encode(str)
	assert.Nil(t, err)
	err = rdb.Decode(encoded, &decoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}
