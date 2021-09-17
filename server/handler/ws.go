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

	"github.com/yhat/wsutil"

	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
)

// HandleWSRequestAndProxy - Handles the websocket requests and proxies to backend server.
func (rc RequestCall) HandleWSRequestAndProxy(ctx context.Context) {
	rc.serveReverseProxyWS(ctx)

	if enableLoggingRequest {
		logger.LogRequest(rc.Request, *rc.Response, rc.ReqID, false, CacheStatusLabel[CacheStatusMiss])
	}
}

func (rc RequestCall) serveReverseProxyWS(ctx context.Context) {
	tracingSpan := tracing.NewChildSpan("handler.serve_reverse_proxy_ws", ctx)
	defer tracingSpan.Finish()

	proxyURL, err := rc.GetUpstreamURL()
	if err != nil {
		rc.GetLogger().Errorf("Cannot process Upstream URL: %s", err.Error())
		return
	}

	rc.GetLogger().Debugf("ProxyURL: %s", proxyURL.String())
	rc.GetLogger().Debugf("Req URL: %s", rc.Request.URL.String())
	rc.GetLogger().Debugf("Req Host: %s", rc.Request.Host)

	telemetry.RegisterRequestUpstream(ctx, proxyURL, enableCachedResponse, CacheStatusLabel[CacheStatusMiss])

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
	proxy.Dial = transport.Dial
	proxy.TLSClientConfig = transport.TLSClientConfig

	proxy.ServeHTTP(rc.Response, &rc.Request)
}
