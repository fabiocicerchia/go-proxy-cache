// +build all unit

package balancer_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

func TestIpHashPickEmpty(t *testing.T) {
	initLogs()

	b := balancer.NewIpHashBalancer("TestIpHashPickEmpty", []balancer.Item{})

	value, err := b.Pick("https://example.com")

	assert.NotNil(t, err)
	assert.Equal(t, "*errors.errorString", fmt.Sprintf("%T", err))
	assert.Equal(t, err.Error(), "no item is available")

	assert.Empty(t, value)
}

func TestIpHashPickWithData(t *testing.T) {
	initLogs()

	b := balancer.NewIpHashBalancer("TestIpHashPickWithData", []balancer.Item{
		balancer.Item{Endpoint: "item1", Healthy: true},
		balancer.Item{Endpoint: "item2", Healthy: true},
		balancer.Item{Endpoint: "item3", Healthy: true},
	})

	value, err := b.Pick("https://example.com")

	assert.Nil(t, err)

	assert.NotNil(t, value)
	assert.NotEmpty(t, value)
	assert.Regexp(t, "^(item1|item2|item3)$", value)
}

func TestIpHashPickCorrectness(t *testing.T) {
	initLogs()

	b := balancer.NewIpHashBalancer("TestIpHashPickCorrectness", []balancer.Item{
		balancer.Item{Endpoint: "item1", Healthy: true},
		balancer.Item{Endpoint: "item2", Healthy: true},
		balancer.Item{Endpoint: "item3", Healthy: true},
	})

	// first round (shuffling)
	var value1, value2, value3, value4 string
	var err error
	value1, err = b.Pick("https://example.com")
	assert.Nil(t, err)
	assert.Regexp(t, "^(item1|item2|item3)$", value1)

	// second round (must be the same)
	value2, err = b.Pick("https://example.com")
	assert.Nil(t, err)
	assert.Equal(t, value1, value2)

	// third round (different key)
	value3, err = b.Pick("https://google.com")
	assert.Nil(t, err)
	assert.Regexp(t, "^(item1|item2|item3)$", value3)

	// fourth round (must be the same)
	value4, err = b.Pick("https://google.com")
	assert.Nil(t, err)
	assert.Equal(t, value3, value4)
}
