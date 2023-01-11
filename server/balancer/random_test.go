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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

func TestRandomPickEmpty(t *testing.T) {
	initLogs()

	b := balancer.NewRandomBalancer("TestRandomPickEmpty", []balancer.Item{})

	value, err := b.Pick("https://example.com")

	assert.NotNil(t, err)
	assert.Equal(t, "*errors.errorString", fmt.Sprintf("%T", err))
	assert.Equal(t, err.Error(), "no item is available")

	assert.Empty(t, value)
}

func TestRandomPickWithData(t *testing.T) {
	initLogs()

	b := balancer.NewRandomBalancer("TestRandomPickWithData", []balancer.Item{
		{Endpoint: "item1", Healthy: true},
		{Endpoint: "item2", Healthy: true},
		{Endpoint: "item3", Healthy: true},
	})

	value, err := b.Pick("https://example.com")

	assert.Nil(t, err)

	assert.NotNil(t, value)
	assert.NotEmpty(t, value)
	assert.Regexp(t, "^(item1|item2|item3)$", value)
}

func TestRandomPickCorrectness(t *testing.T) {
	initLogs()

	b := balancer.NewRandomBalancer("TestRandomPickCorrectness", []balancer.Item{
		{Endpoint: "item1", Healthy: true},
		{Endpoint: "item2", Healthy: true},
		{Endpoint: "item3", Healthy: true},
	})

	// first round (shuffling)
	value1, err := b.Pick("https://example.com")

	assert.Nil(t, err)

	assert.NotNil(t, value1)
	assert.NotEmpty(t, value1)
	assert.Regexp(t, "^(item1|item2|item3)$", value1)

	// second round (random)
	value2, err := b.Pick("https://example.com")

	assert.Nil(t, err)

	assert.NotNil(t, value2)
	assert.NotEmpty(t, value2)
	assert.Regexp(t, "^(item1|item2|item3)$", value2)
}
