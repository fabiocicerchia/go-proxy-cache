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
	"net/url"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

func TestHealthCheckWithCustomPort(t *testing.T) {
	setUp()

	v := &balancer.Item{
		Healthy: false,
		Endpoint: "http://0.0.0.0",
	}
	conf := config.HealthCheck{
		Scheme: "http",
		Port: 8000,
	}
	doHealthCheck(&v, conf)

	assert.EqualTrue(t, v.Healthy)

	tearDown()
}
