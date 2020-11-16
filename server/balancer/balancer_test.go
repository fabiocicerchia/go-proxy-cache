package balancer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

// --- UNIT --------------------------------------------------------------------

func TestGetLBRoundRobinUndefined(t *testing.T) {
	setUp()

	var endpoints []string
	endpoint := balancer.GetLBRoundRobin(endpoints, "8.8.8.8")

	assert.Equal(t, "8.8.8.8", endpoint)

	tearDown()
}

func TestGetLBRoundRobinDefined(t *testing.T) {
	setUp()

	var endpoints = []string{"1.2.3.4"}
	endpoint := balancer.GetLBRoundRobin(endpoints, "8.8.8.8")

	assert.Equal(t, "1.2.3.4", endpoint)

	tearDown()
}

func setUp() {
	config.Config = config.Configuration{}
}

func tearDown() {
	config.Config = config.Configuration{}
}
