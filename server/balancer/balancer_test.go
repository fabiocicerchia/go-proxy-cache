//go:build all || unit
// +build all unit

package balancer_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/url"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

func TestGetUpstreamNodeUndefined(t *testing.T) {
	setUp()

	requestURL, _ := url.Parse("https://example.com")

	conf := config.Upstream{}
	balancer.InitRoundRobin("testing", conf, false)
	endpoint := balancer.GetUpstreamNode("testing", *requestURL, "8.8.8.8")

	assert.Equal(t, "8.8.8.8", endpoint)

	tearDown()
}

func TestGetUpstreamNodeDefined(t *testing.T) {
	setUp()

	requestURL, _ := url.Parse("https://example.com")

	conf := config.Upstream{
		Endpoints: []string{"1.2.3.4"},
	}
	balancer.InitRoundRobin("testing", conf, false)
	endpoint := balancer.GetUpstreamNode("testing", *requestURL, "8.8.8.8")

	assert.Equal(t, "1.2.3.4", endpoint)

	tearDown()
}

func initLogs() {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
}

func setUp() {
	initLogs()

	config.Config = config.Configuration{}
}

func tearDown() {
	config.Config = config.Configuration{}
}
