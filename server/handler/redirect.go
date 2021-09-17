package handler

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/server/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/server/tracing"
)

// RedirectToHTTPS - Redirects from HTTP to HTTPS.
func (rc RequestCall) RedirectToHTTPS(ctx context.Context) {
	targetURL := rc.GetRequestURL()
	targetURL.Scheme = SchemeHTTPS

	rc.GetLogger().Infof("Redirect to: %s", targetURL.String())

	// Just write to client, no need to cache this response.
	http.Redirect(rc.Response.ResponseWriter, &rc.Request, targetURL.String(), rc.DomainConfig.Server.Upstream.RedirectStatusCode)

	tracing.SpanFromContext(ctx).
		SetTag(tracing.TagResponseLocation, targetURL.String()).
		SetTag(tracing.TagResponseStatusCode, rc.DomainConfig.Server.Upstream.RedirectStatusCode)
	metrics.IncStatusCode(rc.DomainConfig.Server.Upstream.RedirectStatusCode)
}
