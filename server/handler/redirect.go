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
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
)

// RedirectToHTTPS - Redirects from HTTP to HTTPS.
func (rc RequestCall) RedirectToHTTPS(ctx context.Context) {
	targetURL := rc.GetRequestURL()
	targetURL.Scheme = SchemeHTTPS

	escapedURL := strings.Replace(targetURL.String(), "\n", "", -1)
	escapedURL = strings.Replace(escapedURL, "\r", "", -1)

	rc.GetLogger().Infof("Redirect to: %s", escapedURL)

	// Just write to client, no need to cache this response.
	http.Redirect(rc.Response.ResponseWriter, &rc.Request, targetURL.String(), rc.DomainConfig.Server.Upstream.RedirectStatusCode)

	telemetry.From(ctx).RegisterRedirect(targetURL)
	telemetry.From(ctx).RegisterStatusCode(rc.DomainConfig.Server.Upstream.RedirectStatusCode)
}
