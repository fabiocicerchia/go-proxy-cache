//go:build all || functional
// +build all functional

package balancer_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

func TestHealthCheckWithCustomPort(t *testing.T) {
	setUpFunctional()

	v := &balancer.Item{
		Healthy: false,
		Endpoint: "http://127.0.0.1",
	}
	conf := config.HealthCheck{
		Scheme: "http",
		Port: "8000",
		StatusCodes: []string{"200"},
	}
	balancer.DoHealthCheck(v, "localhost", conf)

	assert.True(t, v.Healthy)

	tearDownFunctional()
}

func initLogsFunctional() {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
}

func setUpFunctional() {
	initLogsFunctional()

	config.Config = config.Configuration{}
}

func tearDownFunctional() {
	config.Config = config.Configuration{}
}
