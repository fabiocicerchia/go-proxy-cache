// +build all unit

package utils_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"os"
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/stretchr/testify/assert"
)

// --- GetEnv

func TestGetEnvEmptyFallback(t *testing.T) {
	os.Setenv("testing", "1")

	value := utils.GetEnv("testing", "")

	assert.Equal(t, "1", value)

	tearDown()
}

func TestGetEnvFilledFallback(t *testing.T) {
	os.Setenv("testing", "2")

	value := utils.GetEnv("testing", "3")

	assert.Equal(t, "2", value)

	tearDown()
}

func TestGetEnvMissingEnv(t *testing.T) {
	value := utils.GetEnv("testing", "1")

	assert.Equal(t, "1", value)

	tearDown()
}

// --- IfEmpty

func TestIfEmptyWithValue(t *testing.T) {
	value := utils.IfEmpty("text", "fallback")

	assert.Equal(t, "text", value)

	tearDown()
}

func TestIfEmptyWithoutValue(t *testing.T) {
	value := utils.IfEmpty("", "fallback")

	assert.Equal(t, "fallback", value)

	tearDown()
}

// --- Coalesce

func TestCoalesceTrue(t *testing.T) {
	value := "text"
	value = utils.Coalesce(value, "override", value == "").(string)

	assert.Equal(t, "text", value)

	tearDown()
}

func TestCoalesceFalse(t *testing.T) {
	value := ""
	value = utils.Coalesce(value, "override", value == "").(string)

	assert.Equal(t, "override", value)

	tearDown()
}

// --- StripPort

func TestStripPortEmpty(t *testing.T) {
	value := utils.StripPort("")

	assert.Equal(t, "", value)

	tearDown()
}

func TestStripPortHostnameOnly(t *testing.T) {
	value := utils.StripPort("hostname.local")

	assert.Equal(t, "hostname.local", value)

	tearDown()
}

func TestStripPortHostnameAndPort(t *testing.T) {
	value := utils.StripPort("hostname.local:1234")

	assert.Equal(t, "hostname.local", value)

	tearDown()
}

func tearDown() {
	config.Config = config.Configuration{}
	os.Unsetenv("testing")
}
