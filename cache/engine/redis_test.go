// +build functional

package engine_test

import (
	"testing"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerWithPingTimeout1(t *testing.T) {
	config.Config = config.Configuration{
		Cache: config.Cache{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
			CircuitBreaker: config.CircuitBreaker{
				Threshold:   2,   // after 2nd request, if meet FailureRate goes open.
				FailureRate: 0.5, // 1 out of 2 fails, or more
				Interval:    0,
				Timeout:     time.Duration(1), // clears state immediately
			},
		},
	}

	engine.Destroy()

	val := engine.Ping()
	assert.False(t, val)

	engine.InitCircuitBreaker(
		config.Config.Cache.CircuitBreaker.Threshold,
		config.Config.Cache.CircuitBreaker.FailureRate,
		config.Config.Cache.CircuitBreaker.Interval,
		config.Config.Cache.CircuitBreaker.Timeout,
	)
	engine.Connect(config.Config.Cache)

	assert.Equal(t, "closed", engine.CB().State().String())

	val = engine.Ping()
	assert.True(t, val)
	assert.Equal(t, "closed", engine.CB().State().String())

	_ = engine.Close()

	val = engine.Ping()
	assert.False(t, val)
	assert.Equal(t, "closed", engine.CB().State().String())

	val = engine.Ping()
	assert.False(t, val)
	assert.Equal(t, "half-open", engine.CB().State().String())

	engine.Connect(config.Config.Cache)

	val = engine.Ping()
	assert.True(t, val)
	assert.Equal(t, "closed", engine.CB().State().String())
}
