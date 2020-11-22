// +build functional

package engine_test

import (
	"testing"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
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

	engine.Connect(config.Config.Cache)

	assert.Equal(t, "closed", config.CB().State().String())

	val := engine.Ping()
	assert.True(t, val)
	assert.Equal(t, "closed", config.CB().State().String())

	_ = engine.Close()

	val = engine.Ping()
	assert.False(t, val)
	assert.Equal(t, "closed", config.CB().State().String())

	val = engine.Ping()
	assert.False(t, val)
	assert.Equal(t, "half-open", config.CB().State().String())

	engine.Connect(config.Config.Cache)

	val = engine.Ping()
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

	engine.Connect(config.Config.Cache)

	assert.True(t, engine.Ping())

	engine.Close()

	assert.False(t, engine.Ping())
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

	engine.Connect(config.Config.Cache)

	done, err := engine.Set("test", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := engine.Get("test")
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

	engine.Connect(config.Config.Cache)

	done, err := engine.Set("test", "sample", 1*time.Millisecond)
	assert.True(t, done)
	assert.Nil(t, err)

	time.Sleep(10 * time.Millisecond)

	value, err := engine.Get("test")
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

	engine.Connect(config.Config.Cache)

	done, err := engine.Set("test", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := engine.Get("test")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)

	err = engine.Del("test")
	assert.Nil(t, err)

	value, err = engine.Get("test")
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

	engine.Connect(config.Config.Cache)

	done, err := engine.Set("test", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	err = engine.Expire("test", 1*time.Second)
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)

	value, err := engine.Get("test")
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

	engine.Connect(config.Config.Cache)

	err := engine.Push("test", []string{"a", "b", "c"})
	assert.Nil(t, err)

	value, err := engine.List("test")
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

	engine.Connect(config.Config.Cache)

	done, err := engine.Set("test_1", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = engine.Set("test_2", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = engine.Set("test_3", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := engine.Get("test_1")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
	value, err = engine.Get("test_2")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
	value, err = engine.Get("test_3")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)

	len, err := engine.DelWildcard("test_*")
	assert.Equal(t, 3, len)
	assert.Nil(t, err)

	value, err = engine.Get("test_1")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
	value, err = engine.Get("test_2")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
	value, err = engine.Get("test_3")
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

	engine.Connect(config.Config.Cache)

	done, err := engine.Set("test_1", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = engine.Set("test_2", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)
	done, err = engine.Set("test_3", "sample", 0)
	assert.True(t, done)
	assert.Nil(t, err)

	value, err := engine.Get("test_1")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
	value, err = engine.Get("test_2")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)
	value, err = engine.Get("test_3")
	assert.Equal(t, "sample", value)
	assert.Nil(t, err)

	done, err = engine.PurgeAll()
	assert.True(t, done)
	assert.Nil(t, err)

	value, err = engine.Get("test_1")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
	value, err = engine.Get("test_2")
	assert.Equal(t, "", value)
	assert.Equal(t, "redis: nil", err.Error())
	value, err = engine.Get("test_3")
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

	engine.Connect(config.Config.Cache)

	str := []byte("test string")
	var decoded []byte

	encoded, err := engine.Encode(str)
	assert.Nil(t, err)
	err = engine.Decode(encoded, &decoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}
