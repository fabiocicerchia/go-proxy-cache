//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache
package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	log "github.com/sirupsen/logrus"
)

// FixRequest - Fixes the Request in order to use the load balanced host.
func FixRequest(url url.URL, forwarding config.Forward, req *http.Request) {
	scheme := utils.IfEmpty(forwarding.Scheme, url.Scheme)
	host := utils.IfEmpty(forwarding.Host, url.Host)
	port := forwarding.Port
	// TODO: what's fallback?
	// TODO: dups with purge
	if port == "" && scheme == "http" {
		port = "80"
	} else if port == "" && scheme == "https" {
		port = "443"
	}

	req.URL.Host = balancer.GetLBRoundRobin(url.Host) + ":" + port
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
	domainConfig := config.DomainConf(req.Host)

	lwr := response.NewLoggedResponseWriter(res)

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
	if domainConfig == nil {
		// TODO: SERVE A 404?
		log.Errorf("Missing configuration in HandleRequestAndProxy for %s.", req.Host)
		return
	}
	forwarding := domainConfig.Server.Forwarding

	scheme := utils.IfEmpty(forwarding.Scheme, req.URL.Scheme)

	port := forwarding.Port
	// TODO: what's fallback?
	// TODO: dups with purge
	if port == "" && scheme == "http" {
		port = "80"
	} else if port == "" && scheme == "https" {
		port = "443"
	}

	proxyURL := *req.URL
	proxyURL.Scheme = scheme
	// TODO: WHAT IF IT HAS ALREADY PORT?
	proxyURL.Host = forwarding.Host + ":" + port

	cached := serveCachedContent(lwr, *req, proxyURL)
	if !cached {
		serveReverseProxy(forwarding, proxyURL, lwr, req)
	}

	logger.LogRequest(*req, *lwr, cached)
}
