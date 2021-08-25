package circuitbreaker

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/sony/gobreaker"
)

var cb map[string]*gobreaker.CircuitBreaker

// CircuitBreaker - Settings for redis circuit breaker.
type CircuitBreaker struct {
	FailureRate float64
	Interval    time.Duration
	Timeout     time.Duration
	Threshold   uint32
	MaxRequests uint32
}

// InitCircuitBreaker - Initialise the Circuit Breaker.
func InitCircuitBreaker(name string, config CircuitBreaker) {
	st := gobreaker.Settings{
		Name:        name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)

			return counts.Requests >= config.Threshold && failureRatio >= config.FailureRate
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Warnf("Circuit Breaker - Changed from %s to %s", from.String(), to.String())
		},
	}

	if cb == nil {
		cb = make(map[string]*gobreaker.CircuitBreaker)
	}

	cb[name] = gobreaker.NewCircuitBreaker(st)
}

// CB - Returns instance of gobreaker.CircuitBreaker.
func CB(name string) *gobreaker.CircuitBreaker {
	if val, ok := cb[name]; ok {
		return val
	}

	log.Warnf("Missing circuit breaker for %s", name)

	return nil
}
