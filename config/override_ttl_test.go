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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

func TestOverrideTTLMergesPerDomain(t *testing.T) {
	global := config.Configuration{Cache: config.Cache{TTL: 10, OverrideTTL: 60}}

	overridden := global
	overridden.CopyOverWith(config.Configuration{Cache: config.Cache{OverrideTTL: 300}}, nil)
	assert.Equal(t, 300, overridden.Cache.OverrideTTL)
	assert.Equal(t, 10, overridden.Cache.TTL)

	inherited := global
	inherited.CopyOverWith(config.Configuration{Cache: config.Cache{TTL: 20}}, nil)
	assert.Equal(t, 60, inherited.Cache.OverrideTTL)
	assert.Equal(t, 20, inherited.Cache.TTL)
}

func TestOverrideTTLFromEnv(t *testing.T) {
	saved := config.Config
	t.Cleanup(func() { config.Config = saved })

	t.Setenv("OVERRIDE_TTL", "120")
	config.InitConfigFromFileOrEnv(filepath.Join(t.TempDir(), "missing.yml"))

	assert.Equal(t, 120, config.Config.Cache.OverrideTTL)
}

func TestOverrideTTLNegativeRejected(t *testing.T) {
	cases := map[string]string{
		"global":     "cache:\n  override_ttl: -1\n",
		"per domain": "domains:\n  example_com:\n    cache:\n      override_ttl: -1\n",
	}

	for name, yml := range cases {
		file := filepath.Join(t.TempDir(), "config.yml")
		assert.NoError(t, os.WriteFile(file, []byte(yml), 0o600))

		_, err := config.Validate(file)
		assert.ErrorContains(t, err, "override_ttl must be >= 0", name)
	}
}

func TestOverrideTTLValidYAML(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yml")
	assert.NoError(t, os.WriteFile(file, []byte("cache:\n  override_ttl: 300\n"), 0o600))

	_, err := config.Validate(file)
	assert.NoError(t, err)
}
