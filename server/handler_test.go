package server_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server"
)

func TestGetLBRoundRobinUndefined(t *testing.T) {
	setUpHandler()

	var endpoints []string
	endpoint := server.GetLBRoundRobin(endpoints, "8.8.8.8")

	assert.Equal(t, "8.8.8.8", endpoint)

	tearDownHandler()
}

func TestGetLBRoundRobinDefined(t *testing.T) {
	setUpHandler()

	var endpoints = []string{"1.2.3.4"}
	endpoint := server.GetLBRoundRobin(endpoints, "8.8.8.8")

	assert.Equal(t, "1.2.3.4", endpoint)

	tearDownHandler()
}

func setUpHandler() {
	config.Config = config.Configuration{}
}

func tearDownHandler() {
	config.Config = config.Configuration{}
}
