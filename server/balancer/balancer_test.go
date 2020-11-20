// +build unit

package balancer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

func TestGetLBRoundRobinUndefined(t *testing.T) {
	setUp()

	var endpoints []string
	balancer.InitRoundRobin(endpoints)
	endpoint := balancer.GetLBRoundRobin("8.8.8.8")

	assert.Equal(t, "8.8.8.8", endpoint)

	tearDown()
}

func TestGetLBRoundRobinDefined(t *testing.T) {
	setUp()

	var endpoints = []string{"1.2.3.4"}
	balancer.InitRoundRobin(endpoints)
	endpoint := balancer.GetLBRoundRobin("8.8.8.8")

	assert.Equal(t, "1.2.3.4", endpoint)

	tearDown()
}

func setUp() {
	config.Config = config.Configuration{}
}

func tearDown() {
	config.Config = config.Configuration{}
}
