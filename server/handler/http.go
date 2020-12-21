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
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	log "github.com/sirupsen/logrus"
)

var enableStoringResponse = true
var enableCachedResponse = true
var enableLoggingRequest = true

// HandleHTTPRequestAndProxy - Handles the HTTP requests and proxies to backend server.
func (rc RequestCall) HandleHTTPRequestAndProxy(domainConfig *config.Configuration) {
	cached := false

	if enableCachedResponse {
		cached = rc.serveCachedContent()
	}

	if !cached {
		rc.serveReverseProxyHTTP(domainConfig)
	}

	if enableLoggingRequest {
		logger.LogRequest(*rc.Request, *rc.Response, cached)
	}
}

func getOverridePort(host string, port string, scheme string) string {
	// if there's already a port it must have priority
	if strings.Contains(host, ":") {
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

	PushProxiedResources(rc.Response)

	ctx := rc.Request.Context()
	transport.ServeCachedResponse(ctx, rc.Response, uriobj)
	rc.Response.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderHit)

	return true
}

func (rc RequestCall) patchProxyTransport(domainConfig *config.Configuration) *http.Transport {
	// G402 (CWE-295): TLS InsecureSkipVerify may be true. (Confidence: LOW, Severity: HIGH)
	// It can be ignored as it is customisable, but the default is false.
	return &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     1000,
		Dial: func(network, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(network, addr, 15*time.Second)
			if err != nil {
				return conn, err
			}

			return conn, err
		},
		DisableKeepAlives: false,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: domainConfig.Server.Upstream.InsecureBridge,
		},
	} // #nosec
}

func (rc RequestCall) serveReverseProxyHTTP(domainConfig *config.Configuration) {
	upstream := domainConfig.Server.Upstream
	proxyURL := rc.patchRequestForReverseProxy(upstream)

	log.Debugf("ProxyURL: %s", proxyURL.String())
	log.Debugf("Req URL: %s", rc.Request.URL.String())
	log.Debugf("Req Host: %s", rc.Request.Host)

	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	proxy.Transport = rc.patchProxyTransport(domainConfig)

	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		// the default director implementation returned by httputil.NewSingleHostReverseProxy
		// takes care of setting the request Scheme, Host, and Path.
		director(req)

		// TODO: Move patchRequestForReverseProxy in here
	}

	proxy.ServeHTTP(rc.Response, rc.Request)

	if enableStoringResponse {
		rcDTO := ConvertToRequestCallDTO(rc)

		stored, err := storage.StoreGeneratedPage(rcDTO, domainConfig.Cache)
		if !stored || err != nil {
			logger.Log(*rc.Request, fmt.Sprintf("Not Stored: %v", err))
		}
	}
}

// FixRequest - Fixes the Request in order to use the load balanced host.
func (rc *RequestCall) FixRequest(url url.URL, upstream config.Upstream) {
	scheme := utils.IfEmpty(upstream.Scheme, rc.GetScheme())
	host := utils.IfEmpty(upstream.Host, url.Host)

	lbID := upstream.Host + utils.StringSeparatorOne + upstream.Scheme
	balancedHost := balancer.GetLBRoundRobin(lbID, url.Host)
	overridePort := getOverridePort(balancedHost, upstream.Port, scheme)

	// The value of r.URL.Host and r.Host are almost always different. On a
	// proxy server, r.URL.Host is the host of the target server and r.Host is
	// the host of the proxy server itself.
	// Ref: https://stackoverflow.com/a/42926149/888162
	rc.Request.Header.Set("X-Forwarded-Host", rc.Request.Header.Get("Host"))

	rc.Request.URL.Host = balancedHost + overridePort
	rc.Request.URL.Scheme = scheme
	rc.Request.Host = host
}

func (rc *RequestCall) patchRequestForReverseProxy(upstream config.Upstream) *url.URL {
	overridePort := getOverridePort(upstream.Host, upstream.Port, rc.GetScheme())
	targetURL := *rc.Request.URL
	targetURL.Scheme = rc.GetScheme()
	targetURL.Host = upstream.Host + overridePort

	rc.FixRequest(targetURL, upstream)

	proxyURL := &url.URL{
		Scheme: rc.Request.URL.Scheme,
		Host:   rc.Request.URL.Host,
	}

	return proxyURL
}
