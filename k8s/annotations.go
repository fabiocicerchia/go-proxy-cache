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
	"strconv"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
)

// Annotation names, all under AnnotationPrefix.
const (
	AnnCacheEnabled         = "cache-enabled"
	AnnCacheTTL             = "cache-ttl"
	AnnCacheAllowedStatuses = "cache-allowed-statuses"
	AnnCacheAllowedMethods  = "cache-allowed-methods"
	AnnCacheNegativeTTL     = "cache-negative-ttl"
	AnnGZip                 = "gzip"
	AnnCollapsedForwarding  = "collapsed-forwarding"
	AnnBackendProtocol      = "backend-protocol"
	AnnInsecureBridge       = "insecure-bridge"
	AnnBalancingAlgorithm   = "balancing-algorithm"
	AnnPreserveHost         = "preserve-host"
	AnnUpstreamHost         = "upstream-host"
	AnnHTTPToHTTPS          = "http-to-https"
	AnnRedirectStatusCode   = "redirect-status-code"
	AnnHSTSEnabled          = "hsts-enabled"
	AnnHSTSMaxAge           = "hsts-max-age"
	AnnHSTSIncludeSubdomain = "hsts-include-subdomains"
	AnnHSTSPreload          = "hsts-preload"
	AnnPurgeAllowedIPs      = "purge-allowed-ips"
	AnnJwtJwksURL           = "jwt-jwks-url"
	AnnJwtAllowedScopes     = "jwt-allowed-scopes"
	AnnJwtExcludedPaths     = "jwt-excluded-paths"
	AnnRewriteTarget        = "rewrite-target"
	AnnUsePathRegex         = "use-regex"
)

// routeSettings - Per-object overrides parsed from annotations.
//
// The overrides are applied explicitly rather than through
// Configuration.CopyOverWith: that merges with utils.Coalesce, which treats
// every zero value as "not set", so it could never turn a bool off or a
// number down to zero. An annotation that says `gzip: "false"` has to mean
// exactly that.
type routeSettings struct {
	Config config.Configuration

	PreserveHost  bool
	UpstreamHost  string
	BackendScheme string
	RewriteTarget string
	UseRegex      bool
}

// parseAnnotations - Applies annotation overrides on top of a base config.
//
// A malformed value is logged and ignored: rejecting the whole object would
// take a working route out of service over a typo in an unrelated setting.
func parseAnnotations(base config.Configuration, annotations map[string]string, source string) routeSettings {
	s := routeSettings{
		Config:       base,
		PreserveHost: true,
	}

	get := func(name string) (string, bool) {
		v, ok := annotations[AnnotationPrefix+name]

		return strings.TrimSpace(v), ok
	}

	warn := func(name string, value string) {
		logger.GetGlobal().Warnf("Ignoring invalid annotation %s%s=%q on %s", AnnotationPrefix, name, value, source)
	}

	if v, ok := get(AnnCacheEnabled); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			if !b {
				// Nothing is cacheable when no status code is allowed.
				s.Config.Cache.AllowedStatuses = []int{}
				s.Config.Cache.NegativeTTL = nil
			}
		} else {
			warn(AnnCacheEnabled, v)
		}
	}

	if v, ok := get(AnnCacheTTL); ok {
		if ttl, err := strconv.Atoi(v); err == nil {
			s.Config.Cache.TTL = ttl
		} else {
			warn(AnnCacheTTL, v)
		}
	}

	if v, ok := get(AnnCacheAllowedStatuses); ok {
		if statuses, err := parseIntList(v); err == nil {
			s.Config.Cache.AllowedStatuses = statuses
		} else {
			warn(AnnCacheAllowedStatuses, v)
		}
	}

	if v, ok := get(AnnCacheAllowedMethods); ok {
		s.Config.Cache.AllowedMethods = parseStringList(v)
	}

	if v, ok := get(AnnCacheNegativeTTL); ok {
		if m, err := parseIntMap(v); err == nil {
			s.Config.Cache.NegativeTTL = m
		} else {
			warn(AnnCacheNegativeTTL, v)
		}
	}

	applyBool(get, warn, AnnGZip, &s.Config.Server.GZip)
	applyBool(get, warn, AnnCollapsedForwarding, &s.Config.Server.Upstream.CollapsedForwarding)
	applyBool(get, warn, AnnInsecureBridge, &s.Config.Server.Upstream.InsecureBridge)
	applyBool(get, warn, AnnHTTPToHTTPS, &s.Config.Server.Upstream.HTTP2HTTPS)
	applyBool(get, warn, AnnPreserveHost, &s.PreserveHost)
	applyBool(get, warn, AnnUsePathRegex, &s.UseRegex)
	applyBool(get, warn, AnnHSTSEnabled, &s.Config.Server.TLS.HSTS.Enabled)
	applyBool(get, warn, AnnHSTSIncludeSubdomain, &s.Config.Server.TLS.HSTS.IncludeSubdomains)
	applyBool(get, warn, AnnHSTSPreload, &s.Config.Server.TLS.HSTS.Preload)

	applyInt(get, warn, AnnHSTSMaxAge, &s.Config.Server.TLS.HSTS.MaxAge)
	applyInt(get, warn, AnnRedirectStatusCode, &s.Config.Server.Upstream.RedirectStatusCode)

	if v, ok := get(AnnBackendProtocol); ok {
		switch strings.ToLower(v) {
		case "http", "https", "ws", "wss":
			s.BackendScheme = strings.ToLower(v)
		default:
			warn(AnnBackendProtocol, v)
		}
	}

	if v, ok := get(AnnBalancingAlgorithm); ok {
		s.Config.Server.Upstream.BalancingAlgorithm = v
	}

	if v, ok := get(AnnUpstreamHost); ok {
		s.UpstreamHost = v
		s.PreserveHost = false
	}

	if v, ok := get(AnnRewriteTarget); ok {
		s.RewriteTarget = v
	}

	if v, ok := get(AnnPurgeAllowedIPs); ok {
		s.Config.Server.Purge.AllowedIPs = parseStringList(v)
	}

	if v, ok := get(AnnJwtJwksURL); ok && v != "" {
		s.Config.Jwt.JwksUrl = v
		config.InitJWT(&s.Config.Jwt)
	}

	if v, ok := get(AnnJwtAllowedScopes); ok {
		s.Config.Jwt.AllowedScopes = parseStringList(v)
	}

	if v, ok := get(AnnJwtExcludedPaths); ok {
		s.Config.Jwt.ExcludedPaths = parseStringList(v)
	}

	return s
}

type annGetter func(string) (string, bool)
type annWarner func(string, string)

func applyBool(get annGetter, warn annWarner, name string, target *bool) {
	v, ok := get(name)
	if !ok {
		return
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		warn(name, v)
		return
	}

	*target = b
}

func applyInt(get annGetter, warn annWarner, name string, target *int) {
	v, ok := get(name)
	if !ok {
		return
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		warn(name, v)
		return
	}

	*target = n
}

func parseStringList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return out
}

func parseIntList(v string) ([]int, error) {
	parts := parseStringList(v)
	out := make([]int, 0, len(parts))

	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}

		out = append(out, n)
	}

	return out, nil
}

// parseIntMap - Parses "404=30,502=10" into {404: 30, 502: 10}.
func parseIntMap(v string) (map[int]int, error) {
	out := make(map[int]int)

	for _, pair := range parseStringList(v) {
		key, value, found := strings.Cut(pair, "=")
		if !found {
			return nil, strconv.ErrSyntax
		}

		k, err := strconv.Atoi(strings.TrimSpace(key))
		if err != nil {
			return nil, err
		}

		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}

		out[k] = n
	}

	return out, nil
}
