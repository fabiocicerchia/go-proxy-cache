package circuitbreaker

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

var cb map[string]*gobreaker.CircuitBreaker = make(map[string]*gobreaker.CircuitBreaker)

// cbMu - Guards the breaker map, which can be written while requests are in
// flight.
var cbMu sync.RWMutex

// Fallback settings for a breaker requested before one was registered under
// that name. They mirror the defaults in config.Config.CircuitBreaker, which
// this package cannot import (config imports this one).
const (
	defaultThreshold   uint32        = 2
	defaultFailureRate float64       = 0.5
	defaultInterval    time.Duration = 0
	defaultTimeout     time.Duration = 60 * time.Second
	defaultMaxRequests uint32        = 1
)

// CircuitBreaker - Settings for redis circuit breaker.
type CircuitBreaker struct {
	FailureRate float64
	Interval    time.Duration
	Timeout     time.Duration
	Threshold   uint32
	MaxRequests uint32
}

// InitCircuitBreaker - Initialise the Circuit Breaker.
func InitCircuitBreaker(name string, config CircuitBreaker, logger *logrus.Logger) {
	st := gobreaker.Settings{
		Name:          name,
		MaxRequests:   config.MaxRequests,
		Interval:      config.Interval,
		Timeout:       config.Timeout,
		ReadyToTrip:   cbReadyToTrip(config),
		OnStateChange: cbOnStateChange(logger),
	}

	breaker := gobreaker.NewCircuitBreaker(st)

	cbMu.Lock()
	defer cbMu.Unlock()

	cb[name] = breaker
}

func cbReadyToTrip(config CircuitBreaker) func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)

		return counts.Requests >= config.Threshold && failureRatio >= config.FailureRate
	}
}

func cbOnStateChange(log *logrus.Logger) func(name string, from gobreaker.State, to gobreaker.State) {
	return func(name string, from gobreaker.State, to gobreaker.State) {
		log.Warnf("Circuit Breaker '%s' - Changed from %s to %s", name, from.String(), to.String())
	}
}

// CB - Returns instance of gobreaker.CircuitBreaker.
//
// A breaker is created on demand when none is registered under the name. Every
// caller dereferences the result, so returning nil for an unregistered domain
// ID panicked the process from the request path instead of merely losing the
// cache.
func CB(name string, log *logrus.Logger) *gobreaker.CircuitBreaker {
	cbMu.RLock()
	val, ok := cb[name]
	cbMu.RUnlock()

	if ok {
		return val
	}

	log.Warnf("Missing circuit breaker for %s, creating one with default settings", name)

	InitCircuitBreaker(name, CircuitBreaker{
		Threshold:   defaultThreshold,
		FailureRate: defaultFailureRate,
		Interval:    defaultInterval,
		Timeout:     defaultTimeout,
		MaxRequests: defaultMaxRequests,
	}, log)

	cbMu.RLock()
	defer cbMu.RUnlock()

	return cb[name]
}
