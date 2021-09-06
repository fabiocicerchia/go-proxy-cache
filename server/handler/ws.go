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
	"net/http"

	"github.com/yhat/wsutil"

	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
)

// HandleWSRequestAndProxy - Handles the websocket requests and proxies to backend server.
func (rc RequestCall) HandleWSRequestAndProxy() {
	rc.serveReverseProxyWS()

	if enableLoggingRequest {
		logger.LogRequest(rc.Request, *rc.Response, rc.ReqID, false, CacheStatusLabel[CacheStatusMiss])
	}
}

func (rc RequestCall) serveReverseProxyWS() {
	proxyURL := rc.GetUpstreamURL()

	rc.GetLogger().Debugf("ProxyURL: %s", proxyURL.String())
	rc.GetLogger().Debugf("Req URL: %s", rc.Request.URL.String())
	rc.GetLogger().Debugf("Req Host: %s", rc.Request.Host)

	proxy := wsutil.NewSingleHostReverseProxy(&proxyURL)

	originalDirector := proxy.Director
	gpcDirector := rc.ProxyDirector
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
