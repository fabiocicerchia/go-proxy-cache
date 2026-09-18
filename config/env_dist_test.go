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
	"strings"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// .env.dist is meant to be usable as it ships. envconfig rejects an
// empty-but-set value for any type it has to parse -- bool, duration, int --
// and config turns that into a log.Fatal, so one stray `KEY=` keeps the proxy
// from starting at all. Commented-out placeholders are the only safe way to
// document a key with no default.
//
// Asserting on the whole file rather than on one key, because the failure is a
// property of the file and every future addition inherits it.
func TestEnvDistParses(t *testing.T) {
	raw, err := os.ReadFile("../.env.dist")
	assert.Nil(t, err)

	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		t.Setenv(strings.TrimSpace(key), strings.TrimSpace(value))
	}

	var loaded config.Configuration

	assert.Nil(t, envconfig.Process("", &loaded),
		"every uncommented key in .env.dist must carry a parseable value")
}
