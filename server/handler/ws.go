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
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	log "github.com/sirupsen/logrus"
	"github.com/yhat/wsutil"
)

// HandleWSRequestAndProxy - Handles the websocket requests and proxies to backend server.
func (rc RequestCall) HandleWSRequestAndProxy() {
	rc.serveReverseProxyWS()

	if enableLoggingRequest {
		logger.LogRequest(*rc.Request, *rc.Response, false, CacheStatusLabel[CacheStatusMiss])
	}
}

func (rc RequestCall) serveReverseProxyWS() {
	upstream := rc.DomainConfig.Server.Upstream
	proxyURL := rc.patchRequestForReverseProxy(upstream)

	log.Debugf("ProxyURL: %s", proxyURL.String())
	log.Debugf("Req URL: %s", rc.Request.URL.String())
	log.Debugf("Req Host: %s", rc.Request.Host)

	proxy := wsutil.NewSingleHostReverseProxy(proxyURL)

	transport := rc.patchProxyTransport()
	proxy.Dial = transport.Dial
	proxy.TLSClientConfig = transport.TLSClientConfig

	proxy.ServeHTTP(rc.Response, rc.Request)
}
