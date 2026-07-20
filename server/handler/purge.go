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
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/cache"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// isPurgeAuthorized - Checks whether a PURGE request is allowed based on the
// configured access controls (IP allowlist and/or shared secret). When neither is
// configured PURGE is unrestricted; when both are configured both must pass.
func (rc RequestCall) isPurgeAuthorized() bool {
	purge := rc.DomainConfig.Server.Purge

	if purge.Secret != "" {
		header := purge.SecretHeader
		if header == "" {
			header = config.DefaultPurgeSecretHeader
		}

		provided := rc.Request.Header.Get(header)
		// Constant-time comparison to avoid leaking the secret via timing.
		if subtle.ConstantTimeCompare([]byte(provided), []byte(purge.Secret)) != 1 {
			return false
		}
	}

	if len(purge.AllowedIPs) > 0 {
		// Use the direct connection IP, not the spoofable X-Forwarded-For header.
		clientIP := net.ParseIP(utils.StripPort(rc.Request.RemoteAddr))
		if clientIP == nil || !isIPAllowed(clientIP, purge.AllowedIPs) {
			return false
		}
	}

	return true
}

// isIPAllowed - Reports whether ip matches any entry (single IP or CIDR) in the allowlist.
func isIPAllowed(ip net.IP, allowed []string) bool {
	for _, entry := range allowed {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			if _, cidr, err := net.ParseCIDR(entry); err == nil && cidr.Contains(ip) {
				return true
			}
			continue
		}

		if parsed := net.ParseIP(entry); parsed != nil && parsed.Equal(ip) {
			return true
		}
	}

	return false
}

// HandlePurge - Purges the cache for the requested URI.
func (rc RequestCall) HandlePurge(ctx context.Context) {
	if !rc.isPurgeAuthorized() {
		rc.Response.ForceWriteHeader(http.StatusForbidden)
		_ = rc.Response.WriteBody("KO")

		rc.GetLogger().Warnf("Unauthorized PURGE attempt from %s", utils.StripPort(rc.Request.RemoteAddr))

		telemetry.From(ctx).RegisterStatusCode(http.StatusForbidden)

		return
	}

	rcDTO := ConvertToRequestCallDTO(rc)

	status, err := storage.PurgeCachedContent(ctx, rc.DomainConfig.Server.Upstream, rcDTO)
	if !status || err != nil {
		rc.Response.ForceWriteHeader(http.StatusNotFound)
		_ = rc.Response.WriteBody("KO")

		escapedURL := strings.Replace(rc.Request.URL.String(), "\n", "", -1)
		escapedURL = strings.Replace(escapedURL, "\r", "", -1)

		rc.GetLogger().Warnf("URL Not Purged %s: %v\n", escapedURL, err)

		telemetry.From(ctx).RegisterPurge(status, err)
		telemetry.From(ctx).RegisterStatusCode(http.StatusNotFound)

		return
	}

	rc.Response.ForceWriteHeader(http.StatusOK)
	_ = rc.Response.WriteBody("OK")

	telemetry.From(ctx).RegisterPurge(status, nil)
	telemetry.From(ctx).RegisterStatusCode(http.StatusOK)

	if enableLoggingRequest {
		logger.LogRequest(rc.Request, rc.Response.StatusCode, rc.Response.Content.Len(), rc.ReqID, cache.StatusNA)
	}
}
