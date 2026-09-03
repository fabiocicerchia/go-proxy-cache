//go:build all || unit
// +build all unit

package utils_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// --- EscapeLogValue

func TestEscapeLogValueStripsCRLF(t *testing.T) {
	escaped := utils.EscapeLogValue("/path\r\nINFO forged log line")

	assert.Equal(t, "/pathINFO forged log line", escaped)
}

func TestEscapeLogValueLeavesCleanValueAlone(t *testing.T) {
	assert.Equal(t, "/path?a=b", utils.EscapeLogValue("/path?a=b"))
}

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
	value := utils.Coalesce("text", "override").(string)

	assert.Equal(t, "text", value)

	tearDown()
}

func TestCoalesceFalse(t *testing.T) {
	value := utils.Coalesce("", "override").(string)

	assert.Equal(t, "override", value)

	tearDown()
}

// --- IsEmpty

func TestIsEmptyTrue(t *testing.T) {
	assert.True(t, utils.IsEmpty(""))
	assert.True(t, utils.IsEmpty(0))
	assert.True(t, utils.IsEmpty(false))
	assert.True(t, utils.IsEmpty([]int{}))
	assert.True(t, utils.IsEmpty([]string{}))
	assert.True(t, utils.IsEmpty(time.Duration(0)))
	assert.True(t, utils.IsEmpty(nil))
	assert.True(t, utils.IsEmpty((*tls.Config)(nil)))

	tearDown()
}

func TestCoalesceFallsBackOnTypedNilPointer(t *testing.T) {
	fallback := &tls.Config{MinVersion: tls.VersionTLS12}

	value := utils.Coalesce((*tls.Config)(nil), fallback)

	// config.Server.TLS.Override is a *tls.Config: returning the typed nil
	// here wiped the built-in TLS defaults and crashed the HTTPS listener.
	assert.Same(t, fallback, value)

	tearDown()
}

func TestIsEmptyFalse(t *testing.T) {
	assert.False(t, utils.IsEmpty("qwerty"))
	assert.False(t, utils.IsEmpty(1))
	assert.False(t, utils.IsEmpty(true))
	assert.False(t, utils.IsEmpty([]int{1, 2, 3}))
	assert.False(t, utils.IsEmpty([]string{"a", "b", "c"}))
	assert.False(t, utils.IsEmpty(time.Duration(1)))

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

func TestStripPortIPv6WithPort(t *testing.T) {
	value := utils.StripPort("[::1]:1234")

	assert.Equal(t, "::1", value)

	tearDown()
}

func TestStripPortIPv6WithBrackets(t *testing.T) {
	value := utils.StripPort("[::1]")

	assert.Equal(t, "::1", value)

	tearDown()
}

func TestStripPortIPv6WithoutPort(t *testing.T) {
	value := utils.StripPort("::1")

	assert.Equal(t, "::1", value)

	tearDown()
}

func tearDown() {
	config.Config = config.Configuration{}
	os.Unsetenv("testing")
}
