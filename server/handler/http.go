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
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
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

// For server requests the URL is parsed from the URI supplied on the
// Request-Line as stored in RequestURI. For most requests, fields other than
// Path and RawQuery will be empty. (See RFC 7230, Section 5.3)
// Ref: https://github.com/golang/go/issues/28940
func getSchemeFromRequest(req http.Request) string {
	if req.TLS != nil {
		// TODO: COVERAGE
		return "https"
	}
	return "http"
}

// FixRequest - Fixes the Request in order to use the load balanced host.
func FixRequest(url url.URL, forwarding config.Forward, req *http.Request) {
	scheme := utils.IfEmpty(forwarding.Scheme, getSchemeFromRequest(*req))
	host := utils.IfEmpty(forwarding.Host, url.Host)

	balancedHost := balancer.GetLBRoundRobin(forwarding.Host, url.Host)
	overridePort := getOverridePort(balancedHost, forwarding.Port, scheme)

	// The value of r.URL.Host and r.Host are almost always different. On a
	// proxy server, r.URL.Host is the host of the target server and r.Host is
	// the host of the proxy server itself.
	// Ref: https://stackoverflow.com/a/42926149/888162
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

	req.URL.Host = balancedHost + overridePort
	req.URL.Scheme = scheme
	req.Host = host
}

func serveReverseProxy(
	forwarding config.Forward,
	target url.URL,
	lwr *response.LoggedResponseWriter,
	req *http.Request,
) {
	domainConfig := config.DomainConf(req.Host)

	FixRequest(target, forwarding, req)

	proxyURL := &url.URL{
		Scheme: req.URL.Scheme,
		Host:   req.URL.Host,
	}

	log.Debugf("ProxyURL: %s", proxyURL.String())
	log.Debugf("Req URL: %s", req.URL.String())
	log.Debugf("Req Host: %s", req.Host)

	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: domainConfig.Server.Forwarding.InsecureBridge,
		},
	}
	proxy.ServeHTTP(lwr, req)

	stored, err := storage.StoreGeneratedPage(*req, *lwr)
	if !stored || err != nil {
		logger.Log(*req, fmt.Sprintf("Not Stored: %v", err))
	}
}

func serveCachedContent(
	lwr *response.LoggedResponseWriter,
	req http.Request,
	url url.URL,
) bool {
	uriobj, err := storage.RetrieveCachedContent(lwr, req)
	if err != nil {
		lwr.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderMiss)

		log.Warnf("Error on serving cached content: %s", err)
		return false
	}

	ctx := req.Context()
	transport.ServeCachedResponse(ctx, lwr, uriobj, uriobj.URL)
	lwr.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderHit)

	return true
}

func getListeningPort(ctx context.Context) string {
	localAddrContextKey := ctx.Value(http.LocalAddrContextKey)
	listeningPort := ""
	if localAddrContextKey != nil {
		srvAddr := localAddrContextKey.(*net.TCPAddr)
		listeningPort = strconv.Itoa(srvAddr.Port)
	}

	return listeningPort
}

// HandleRequest - Handles the entrypoint and directs the traffic to the right handler.
func HandleRequest(res http.ResponseWriter, req *http.Request) {
	lwr := response.NewLoggedResponseWriter(res)

	listeningPort := getListeningPort(req.Context())

	domainConfig := config.DomainConf(req.Host)
	if domainConfig == nil ||
		(domainConfig.Server.Port.HTTP != listeningPort &&
			domainConfig.Server.Port.HTTPS != listeningPort) {
		lwr.WriteHeader(http.StatusNotImplemented)
		logger.LogRequest(*req, *lwr, false)
		log.Errorf("Missing configuration in HandleRequest for %s (listening on :%s).", req.Host, listeningPort)
		return
	}

	if getSchemeFromRequest(*req) == "http" && domainConfig.Server.Forwarding.HTTP2HTTPS {
		RedirectToHTTPS(lwr.ResponseWriter, req, domainConfig.Server.Forwarding.RedirectStatusCode)
		return
	}

	if req.Method == "PURGE" {
		HandlePurge(lwr, req)
		return
	}

	if req.Method == http.MethodConnect {
		lwr.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		HandleRequestAndProxy(lwr, req)
	}
}

// HandleRequestAndProxy - Handles the requests and proxies to backend server.
func HandleRequestAndProxy(lwr *response.LoggedResponseWriter, req *http.Request) {
	domainConfig := config.DomainConf(req.Host)
	forwarding := domainConfig.Server.Forwarding

	scheme := utils.IfEmpty(forwarding.Scheme, getSchemeFromRequest(*req))
	overridePort := getOverridePort(forwarding.Host, forwarding.Port, scheme)

	proxyURL := *req.URL
	proxyURL.Scheme = scheme
	proxyURL.Host = forwarding.Host + overridePort

	cached := serveCachedContent(lwr, *req, proxyURL)
	if !cached {
		serveReverseProxy(forwarding, proxyURL, lwr, req)
	}

	logger.LogRequest(*req, *lwr, cached)
}
