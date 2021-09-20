//go:build all || unit
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

func TestRoundRobinPickEmpty(t *testing.T) {
	initLogs()

	b := balancer.NewRoundRobinBalancer("TestRoundRobinPickEmpty", []balancer.Item{})

	value, err := b.Pick("https://example.com")

	assert.NotNil(t, err)
	assert.Equal(t, "*errors.errorString", fmt.Sprintf("%T", err))
	assert.Equal(t, err.Error(), "no item is available")

	assert.Empty(t, value)
}

func TestRoundRobinPickWithData(t *testing.T) {
	initLogs()

	b := balancer.NewRoundRobinBalancer("TestRoundRobinPickWithData", []balancer.Item{
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

func TestRoundRobinPickCorrectness(t *testing.T) {
	initLogs()

	b := balancer.NewRoundRobinBalancer("TestRoundRobinPickCorrectness", []balancer.Item{
		{Endpoint: "item1", Healthy: true},
		{Endpoint: "item2", Healthy: true},
		{Endpoint: "item3", Healthy: true},
	})

	// first round (shuffling)
	var value1, value2, value3, value4 string
	value1, err := b.Pick("https://example.com")
	assert.Nil(t, err)
	assert.Regexp(t, "^(item1|item2|item3)$", value1)

	// second round (sequential)
	switch value1 {
	case "item1":
		value2, err = b.Pick("https://example.com")
		assert.Nil(t, err)
		assert.Equal(t, "item2", value2)
	case "item2":
		value2, err = b.Pick("https://example.com")
		assert.Nil(t, err)
		assert.Equal(t, "item3", value2)
	case "item3":
		value2, err = b.Pick("https://example.com")
		assert.Nil(t, err)
		assert.Equal(t, "item1", value2)
	}

	// third round (sequential)
	switch value2 {
	case "item1":
		value3, err = b.Pick("https://example.com")
		assert.Nil(t, err)
		assert.Equal(t, "item2", value3)
	case "item2":
		value3, err = b.Pick("https://example.com")
		assert.Nil(t, err)
		assert.Equal(t, "item3", value3)
	case "item3":
		value3, err = b.Pick("https://example.com")
		assert.Nil(t, err)
		assert.Equal(t, "item1", value3)
	}

	// fourth round (sequential)
	switch value3 {
	case "item1":
		value4, err = b.Pick("https://example.com")
		assert.Nil(t, err)
		assert.Equal(t, "item2", value4)
	case "item2":
		value4, err = b.Pick("https://example.com")
		assert.Nil(t, err)
		assert.Equal(t, "item3", value4)
	case "item3":
		value4, err = b.Pick("https://example.com")
		assert.Nil(t, err)
		assert.Equal(t, "item1", value4)
	}
}
