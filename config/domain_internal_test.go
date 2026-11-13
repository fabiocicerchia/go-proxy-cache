//go:build all || unit
// +build all unit

package config

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
)

// Every distinct "host:<port>" variant used to add a new domainsCache entry,
// letting any client grow memory without bound via the Host header.
func TestDomainConfCacheBoundedByHostPortVariants(t *testing.T) {
	c := &Configuration{
		Domains: Domains{
			"example": Configuration{Server: Server{Upstream: Upstream{Host: "example.com", Scheme: "https"}}},
		},
	}

	for i := 0; i < 1000; i++ {
		conf, found := c.DomainConf(fmt.Sprintf("example.com:%d", i), "https")
		assert.True(t, found)
		assert.Equal(t, "example.com", conf.Server.Upstream.Host)
	}
	conf, found := c.DomainConf("example.com:not-a-port", "https")
	assert.True(t, found)
	assert.Equal(t, "example.com", conf.Server.Upstream.Host)

	assert.Len(t, c.domainsCache, 1)
}
