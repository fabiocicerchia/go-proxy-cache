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
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

func FixRequest(url url.URL, forwarding config.Forward, req *http.Request) {
	// TODO: COVERAGE (Test roundrobin)
	scheme := utils.IfEmpty(forwarding.Scheme, url.Scheme)
	host := utils.IfEmpty(forwarding.Host, url.Host)

	req.URL.Host = balancer.GetLBRoundRobin(forwarding.Endpoints, url.Host)
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

	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	proxy.ServeHTTP(lwr, req)

	stored, err := storage.StoreGeneratedPage(*req, *lwr)
	if !stored || err != nil {
		logger.Log(*req, fmt.Sprintf("Not Stored: %v", err))
	}
}

func HandleRequest(res http.ResponseWriter, req *http.Request) {
	lwr := response.NewLoggedResponseWriter(res)

	if config.Config.Server.Forwarding.HTTP2HTTPS {
		// TODO: COVERAGE
		RedirectToHTTPS(lwr.ResponseWriter, req, config.Config.Server.Forwarding.RedirectStatusCode)
		return
	}

	if req.Method == "PURGE" {
		HandlePurge(lwr, req)
		return
	}

	if req.Method == http.MethodConnect {
		// TODO: COVERAGE
		lwr.WriteHeader(http.StatusMethodNotAllowed)
		// HandleTunneling(lwr, req)
	} else {
		HandleRequestAndProxy(lwr, req)
	}
}

// HandleRequestAndProxy - Handles the requests and proxies to backend server.
func HandleRequestAndProxy(lwr *response.LoggedResponseWriter, req *http.Request) {
	forwarding := config.GetForwarding()

	scheme := utils.IfEmpty(forwarding.Scheme, req.URL.Scheme)

	proxyURL := *req.URL
	proxyURL.Scheme = scheme
	proxyURL.Host = forwarding.Host

	cached := storage.ServeCachedContent(lwr, *req, proxyURL)
	if !cached {
		serveReverseProxy(forwarding, proxyURL, lwr, req)
	}

	logger.LogRequest(*req, *lwr, cached)
}
