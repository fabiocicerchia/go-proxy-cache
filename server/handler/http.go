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

// FixRequest - Fixes the Request in order to use the load balanced host.
func FixRequest(url url.URL, forwarding config.Forward, req *http.Request) {
	scheme := utils.IfEmpty(forwarding.Scheme, url.Scheme)
	host := utils.IfEmpty(forwarding.Host, url.Host)

	balancedHost := balancer.GetLBRoundRobin(url.Host)
	overridePort := getOverridePort(balancedHost, forwarding.Port, scheme)

	req.URL.Host = balancedHost + overridePort
	req.URL.Scheme = scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = host
}

func serveReverseProxy(
	forwarding config.Forward,
	target url.URL,
	lwr *response.LoggedResponseWriter,
	req *http.Request,
) {
	FixRequest(target, forwarding, req)

	proxyURL := &url.URL{
		Scheme: target.Scheme,
		Host:   target.Host,
	}

	log.Debugf("ProxyURL: %s", proxyURL.String())
	log.Debugf("Req URL: %s", req.URL.String())
	log.Debugf("Req Host: %s", req.Host)

	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
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

// HandleRequest - Handles the entrypoint and directs the traffic to the right handler.
func HandleRequest(res http.ResponseWriter, req *http.Request) {
	lwr := response.NewLoggedResponseWriter(res)

	ctx := req.Context()
	localAddrContextKey := ctx.Value(http.LocalAddrContextKey)
	listeningPort := ""
	if localAddrContextKey != nil {
		srvAddr := localAddrContextKey.(*net.TCPAddr)
		listeningPort = strconv.Itoa(srvAddr.Port)
	}

	domainConfig := config.DomainConf(req.Host)
	if domainConfig == nil ||
		(domainConfig.Server.Port.HTTP != listeningPort &&
			domainConfig.Server.Port.HTTPS != listeningPort) {
		lwr.WriteHeader(http.StatusNotImplemented)
		logger.LogRequest(*req, *lwr, false)
		log.Errorf("Missing configuration in HandleRequest for %s (listening on :%s).", req.Host, listeningPort)
		return
	}

	if req.URL.Scheme == "http" && domainConfig.Server.Forwarding.HTTP2HTTPS {
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

	scheme := utils.IfEmpty(forwarding.Scheme, req.URL.Scheme)
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
