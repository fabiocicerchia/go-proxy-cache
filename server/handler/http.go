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
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	log "github.com/sirupsen/logrus"
)

// HandleRequestAndProxy - Handles the requests and proxies to backend server.
func (rc RequestCall) HandleRequestAndProxy(domainConfig *config.Configuration) {
	cached := rc.serveCachedContent()
	if !cached {
		rc.serveReverseProxy(domainConfig)
	}

	logger.LogRequest(*rc.Request, *rc.Response, cached)
}

func getOverridePort(host string, port string, scheme string) string {
	// if there's already a port it must have priority
	if strings.Contains(host, ":") {
		// TODO: COVERAGE
		return ""
	}

	portOverride := port

	if portOverride == "" && scheme == "http" {
		portOverride = "80"
	} else if portOverride == "" && scheme == "https" {
		portOverride = "443"
	}

	if portOverride != "" {
		portOverride = ":" + portOverride
	}

	return portOverride
}

func (rc RequestCall) serveCachedContent() bool {
	rcDTO := ConvertToRequestCallDTO(rc)

	uriobj, err := storage.RetrieveCachedContent(rcDTO)
	if err != nil {
		rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderMiss)

		log.Warnf("Error on serving cached content: %s", err)
		return false
	}

	ctx := rc.Request.Context()
	transport.ServeCachedResponse(ctx, rc.Response, uriobj)
	rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderHit)

	return true
}

func (rc RequestCall) serveReverseProxy(domainConfig *config.Configuration) {
	forwarding := domainConfig.Server.Forwarding
	proxyURL := rc.patchRequestForReverseProxy(forwarding)

	log.Debugf("ProxyURL: %s", proxyURL.String())
	log.Debugf("Req URL: %s", rc.Request.URL.String())
	log.Debugf("Req Host: %s", rc.Request.Host)

	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	// G402 (CWE-295): TLS InsecureSkipVerify may be true. (Confidence: LOW, Severity: HIGH)
	// It can be ignored as it is customisable, but the default is false.
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: domainConfig.Server.Forwarding.InsecureBridge,
		},
	} // #nosec
	proxy.ServeHTTP(rc.Response, rc.Request)

	rcDTO := ConvertToRequestCallDTO(rc)

	stored, err := storage.StoreGeneratedPage(rcDTO, domainConfig.Cache)
	if !stored || err != nil {
		logger.Log(*rc.Request, fmt.Sprintf("Not Stored: %v", err))
	}
}

// FixRequest - Fixes the Request in order to use the load balanced host.
func (rc *RequestCall) FixRequest(url url.URL, forwarding config.Forward) {
	scheme := utils.IfEmpty(forwarding.Scheme, rc.GetScheme())
	host := utils.IfEmpty(forwarding.Host, url.Host)

	balancedHost := balancer.GetLBRoundRobin(forwarding.Host, url.Host)
	overridePort := getOverridePort(balancedHost, forwarding.Port, scheme)

	// The value of r.URL.Host and r.Host are almost always different. On a
	// proxy server, r.URL.Host is the host of the target server and r.Host is
	// the host of the proxy server itself.
	// Ref: https://stackoverflow.com/a/42926149/888162
	rc.Request.Header.Set("X-Forwarded-Host", rc.Request.Header.Get("Host"))

	rc.Request.URL.Host = balancedHost + overridePort
	rc.Request.URL.Scheme = scheme
	rc.Request.Host = host
}

func (rc *RequestCall) patchRequestForReverseProxy(forwarding config.Forward) *url.URL {
	overridePort := getOverridePort(forwarding.Host, forwarding.Port, rc.GetScheme())
	targetURL := *rc.Request.URL
	targetURL.Scheme = rc.GetScheme()
	targetURL.Host = forwarding.Host + overridePort

	rc.FixRequest(targetURL, forwarding)

	proxyURL := &url.URL{
		Scheme: rc.Request.URL.Scheme,
		Host:   rc.Request.URL.Host,
	}

	return proxyURL
}
