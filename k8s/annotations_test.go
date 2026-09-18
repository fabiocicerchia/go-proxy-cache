//go:build all || unit
// +build all unit

package k8s

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

func annotations(pairs ...string) map[string]string {
	out := make(map[string]string, len(pairs)/2)

	for i := 0; i+1 < len(pairs); i += 2 {
		out[AnnotationPrefix+pairs[i]] = pairs[i+1]
	}

	return out
}

func TestAnnotationsDefaults(t *testing.T) {
	s := parseAnnotations(config.Config, nil, "test")

	assert.True(t, s.PreserveHost)
	assert.Equal(t, "", s.BackendScheme)
	assert.Equal(t, config.Config.Cache.TTL, s.Config.Cache.TTL)
}

func TestAnnotationsCanTurnBooleansOff(t *testing.T) {
	base := config.Config
	base.Server.GZip = true
	base.Server.Upstream.CollapsedForwarding = true

	s := parseAnnotations(base, annotations(
		AnnGZip, "false",
		AnnCollapsedForwarding, "false",
	), "test")

	// The configuration merger coalesces zero values away, so it could never
	// express this. Annotations have to be applied explicitly.
	assert.False(t, s.Config.Server.GZip)
	assert.False(t, s.Config.Server.Upstream.CollapsedForwarding)
}

func TestAnnotationsCacheSettings(t *testing.T) {
	s := parseAnnotations(config.Config, annotations(
		AnnCacheTTL, "300",
		AnnCacheAllowedStatuses, "200,301,404",
		AnnCacheAllowedMethods, "GET, HEAD",
		AnnCacheNegativeTTL, "404=30, 502=10",
	), "test")

	assert.Equal(t, 300, s.Config.Cache.TTL)
	assert.Equal(t, []int{200, 301, 404}, s.Config.Cache.AllowedStatuses)
	assert.Equal(t, []string{"GET", "HEAD"}, s.Config.Cache.AllowedMethods)
	assert.Equal(t, map[int]int{404: 30, 502: 10}, s.Config.Cache.NegativeTTL)
}

func TestAnnotationsCacheDisabled(t *testing.T) {
	base := config.Config
	base.Cache.NegativeTTL = map[int]int{404: 30}

	s := parseAnnotations(base, annotations(AnnCacheEnabled, "false"), "test")

	assert.Empty(t, s.Config.Cache.AllowedStatuses)
	assert.Nil(t, s.Config.Cache.NegativeTTL)
}

func TestAnnotationsInvalidValuesAreIgnored(t *testing.T) {
	base := config.Config
	base.Cache.TTL = 42

	s := parseAnnotations(base, annotations(
		AnnCacheTTL, "not-a-number",
		AnnGZip, "maybe",
		AnnBackendProtocol, "gopher",
		AnnCacheNegativeTTL, "404",
	), "test")

	// A typo in one annotation must never take the route out of service.
	assert.Equal(t, 42, s.Config.Cache.TTL)
	assert.Equal(t, base.Server.GZip, s.Config.Server.GZip)
	assert.Equal(t, "", s.BackendScheme)
	assert.Equal(t, base.Cache.NegativeTTL, s.Config.Cache.NegativeTTL)
}

func TestAnnotationsBackendProtocol(t *testing.T) {
	s := parseAnnotations(config.Config, annotations(AnnBackendProtocol, "HTTPS"), "test")

	assert.Equal(t, "https", s.BackendScheme)
}

func TestAnnotationsUpstreamHostImpliesNoPreserveHost(t *testing.T) {
	s := parseAnnotations(config.Config, annotations(AnnUpstreamHost, "backend.internal"), "test")

	assert.Equal(t, "backend.internal", s.UpstreamHost)
	assert.False(t, s.PreserveHost)
}

func TestAnnotationsHSTSAndPurge(t *testing.T) {
	s := parseAnnotations(config.Config, annotations(
		AnnHSTSEnabled, "true",
		AnnHSTSMaxAge, "600",
		AnnHSTSIncludeSubdomain, "true",
		AnnHSTSPreload, "true",
		AnnPurgeAllowedIPs, "10.0.0.0/8, 192.168.1.1",
	), "test")

	assert.True(t, s.Config.Server.TLS.HSTS.Enabled)
	assert.Equal(t, 600, s.Config.Server.TLS.HSTS.MaxAge)
	assert.True(t, s.Config.Server.TLS.HSTS.IncludeSubdomains)
	assert.True(t, s.Config.Server.TLS.HSTS.Preload)
	assert.Equal(t, []string{"10.0.0.0/8", "192.168.1.1"}, s.Config.Server.Purge.AllowedIPs)
}

func TestParseIntMap(t *testing.T) {
	m, err := parseIntMap("404=30,502=10")
	assert.NoError(t, err)
	assert.Equal(t, map[int]int{404: 30, 502: 10}, m)

	_, err = parseIntMap("404")
	assert.Error(t, err)

	_, err = parseIntMap("abc=1")
	assert.Error(t, err)
}
