//go:build all || unit
// +build all unit

package handler

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

func TestIsIPAllowed(t *testing.T) {
	assert.True(t, isIPAllowed(net.ParseIP("10.0.0.5"), []string{"10.0.0.0/24"}))
	assert.False(t, isIPAllowed(net.ParseIP("10.0.1.5"), []string{"10.0.0.0/24"}))

	assert.True(t, isIPAllowed(net.ParseIP("192.168.1.1"), []string{"192.168.1.1"}))
	assert.False(t, isIPAllowed(net.ParseIP("192.168.1.2"), []string{"192.168.1.1"}))

	assert.True(t, isIPAllowed(net.ParseIP("::1"), []string{"::1"}))

	assert.False(t, isIPAllowed(net.ParseIP("10.0.0.5"), []string{}))

	// Invalid entries are skipped, valid ones still match.
	assert.True(t, isIPAllowed(net.ParseIP("10.0.0.5"), []string{"not-an-ip", " 10.0.0.5 "}))
}

func TestIsPurgeAuthorizedUnrestricted(t *testing.T) {
	rc := RequestCall{DomainConfig: config.Configuration{}}
	rc.Request = http.Request{RemoteAddr: "1.2.3.4:5678", Header: http.Header{}}

	assert.True(t, rc.isPurgeAuthorized())
}

func TestIsPurgeAuthorizedSecret(t *testing.T) {
	rc := RequestCall{DomainConfig: config.Configuration{}}
	rc.DomainConfig.Server.Purge = config.Purge{Secret: "s3cr3t"}
	rc.Request = http.Request{RemoteAddr: "1.2.3.4:5678", Header: http.Header{}}

	// Missing secret header -> denied.
	assert.False(t, rc.isPurgeAuthorized())

	// Wrong secret -> denied.
	rc.Request.Header.Set(config.DefaultPurgeSecretHeader, "nope")
	assert.False(t, rc.isPurgeAuthorized())

	// Correct secret -> authorized.
	rc.Request.Header.Set(config.DefaultPurgeSecretHeader, "s3cr3t")
	assert.True(t, rc.isPurgeAuthorized())
}

func TestIsPurgeAuthorizedSecretCustomHeader(t *testing.T) {
	rc := RequestCall{DomainConfig: config.Configuration{}}
	rc.DomainConfig.Server.Purge = config.Purge{Secret: "s3cr3t", SecretHeader: "X-Custom-Purge"}
	rc.Request = http.Request{RemoteAddr: "1.2.3.4:5678", Header: http.Header{}}

	// Default header is not honored when a custom one is configured.
	rc.Request.Header.Set(config.DefaultPurgeSecretHeader, "s3cr3t")
	assert.False(t, rc.isPurgeAuthorized())

	rc.Request.Header.Set("X-Custom-Purge", "s3cr3t")
	assert.True(t, rc.isPurgeAuthorized())
}

func TestIsPurgeAuthorizedIPAllowlist(t *testing.T) {
	rc := RequestCall{DomainConfig: config.Configuration{}}
	rc.DomainConfig.Server.Purge = config.Purge{AllowedIPs: []string{"1.2.3.0/24"}}
	rc.Request = http.Request{Header: http.Header{}}

	rc.Request.RemoteAddr = "1.2.3.4:5678"
	assert.True(t, rc.isPurgeAuthorized())

	rc.Request.RemoteAddr = "9.9.9.9:1234"
	assert.False(t, rc.isPurgeAuthorized())
}

func TestIsPurgeAuthorizedBothMustPass(t *testing.T) {
	rc := RequestCall{DomainConfig: config.Configuration{}}
	rc.DomainConfig.Server.Purge = config.Purge{
		AllowedIPs: []string{"1.2.3.0/24"},
		Secret:     "s3cr3t",
	}
	rc.Request = http.Request{RemoteAddr: "1.2.3.4:5678", Header: http.Header{}}

	// IP allowed but secret missing -> denied.
	assert.False(t, rc.isPurgeAuthorized())

	// Secret provided but IP not allowed -> denied.
	rc.Request.Header.Set(config.DefaultPurgeSecretHeader, "s3cr3t")
	rc.Request.RemoteAddr = "9.9.9.9:1234"
	assert.False(t, rc.isPurgeAuthorized())

	// Both pass -> authorized.
	rc.Request.RemoteAddr = "1.2.3.4:5678"
	assert.True(t, rc.isPurgeAuthorized())
}
