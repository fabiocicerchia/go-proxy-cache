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
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

var cb map[string]*gobreaker.CircuitBreaker = make(map[string]*gobreaker.CircuitBreaker)

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

	cb[name] = gobreaker.NewCircuitBreaker(st)
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
func CB(name string, log *logrus.Logger) *gobreaker.CircuitBreaker {
	if val, ok := cb[name]; ok {
		return val
	}

	log.Warnf("Missing circuit breaker for %s", name)

	return nil
}
