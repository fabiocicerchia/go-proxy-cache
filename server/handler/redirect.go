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
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// RedirectToHTTPS - Redirects from HTTP to HTTPS.
//
// The redirect target's host is the request's own Host header, which static
// analysis reads as an attacker-controlled redirect destination (gosec G710,
// CodeQL go/open-redirect). It is not one: HandleRequest calls
// initRequestParams before it reaches here, and that rejects with 501 any
// Host that config.DomainConf does not resolve to a configured domain. Only a
// host already on the configured allowlist can arrive at this line.
//
// Keep that ordering. If the DomainConf check ever moves after this call, the
// finding becomes real.
func (rc RequestCall) RedirectToHTTPS(ctx context.Context) {
	targetURL := rc.GetRequestURL()
	targetURL.Scheme = SchemeHTTPS

	escapedURL := utils.EscapeLogValue(targetURL.String())

	rc.GetLogger().Infof("Redirect to: %s", escapedURL)

	// Just write to client, no need to cache this response.
	http.Redirect(rc.Response.ResponseWriter, &rc.Request, targetURL.String(), rc.DomainConfig.Server.Upstream.RedirectStatusCode)

	telemetry.From(ctx).RegisterRedirect(targetURL)
	telemetry.From(ctx).RegisterStatusCode(rc.DomainConfig.Server.Upstream.RedirectStatusCode)
}
