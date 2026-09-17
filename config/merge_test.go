//go:build all || unit
// +build all unit

package config_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// TestTLSOverrideSurvivesMerge - server/tls.newDefaultTLSConfig dereferences
// config.Server.TLS.Override unconditionally, so a nil there is a startup
// SIGSEGV on every HTTPS listener.
func TestTLSOverrideSurvivesMerge(t *testing.T) {
	config.InitConfigFromFileOrEnv("../test/full-setup/config.yml")

	assert.NotNil(t, config.Config.Server.TLS.Override)

	for name, domain := range config.Config.Domains {
		assert.NotNil(t, domain.Server.TLS.Override, "domain %s", name)
	}
}
