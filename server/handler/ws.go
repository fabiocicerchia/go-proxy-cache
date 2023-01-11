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

	"github.com/yhat/wsutil"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/cache"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
)

// HandleWSRequestAndProxy - Handles the websocket requests and proxies to backend server.
func (rc RequestCall) HandleWSRequestAndProxy(ctx context.Context) {
	rc.serveReverseProxyWS(ctx)

	if enableLoggingRequest {
		logger.LogRequest(rc.Request, rc.Response.StatusCode, rc.Response.Content.Len(), rc.ReqID, cache.StatusMiss)
	}
}

func (rc RequestCall) serveReverseProxyWS(ctx context.Context) {
	tracingSpan := tracing.NewChildSpan(ctx, "handler.serve_reverse_proxy_ws")
	defer tracingSpan.Finish()

	proxyURL, err := rc.GetUpstreamURL()
	if err != nil {
		rc.GetLogger().Errorf("Cannot process Upstream URL: %s", err.Error())
		return
	}

	escapedURL := strings.Replace(rc.Request.URL.String(), "\n", "", -1)
	escapedURL = strings.Replace(escapedURL, "\r", "", -1)

	rc.GetLogger().Debugf("ProxyURL: %s", proxyURL.String())
	rc.GetLogger().Debugf("Req URL: %s", escapedURL)
	rc.GetLogger().Debugf("Req Host: %s", rc.Request.Host)

	telemetry.From(ctx).RegisterRequestUpstream(proxyURL, enableCachedResponse, cache.StatusLabel[cache.StatusMiss])

	proxy := wsutil.NewSingleHostReverseProxy(&proxyURL)

	originalDirector := proxy.Director
	gpcDirector := rc.ProxyDirector(tracingSpan)
	proxy.Director = func(req *http.Request) {
		// the default director implementation returned by httputil.NewSingleHostReverseProxy
		// takes care of setting the request Scheme, Host, and Path.
		originalDirector(req)
		gpcDirector(req)
	}

	transport := rc.patchProxyTransport()
	proxy.Dial = transport.Dial //nolint:staticcheck SA1019
	proxy.TLSClientConfig = transport.TLSClientConfig

	proxy.ServeHTTP(rc.Response, &rc.Request)
}
