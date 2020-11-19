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

func fixRequest(url url.URL, forwarding config.Forward, req *http.Request) {
	scheme := utils.IfEmpty(forwarding.Scheme, url.Scheme)
	host := utils.IfEmpty(forwarding.Host, url.Host)

	req.URL.Host = balancer.GetLBRoundRobin(forwarding.Endpoints, url.Host)
	req.URL.Scheme = scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = host
}

func serveReverseProxy(
	forwarding config.Forward,
	target string,
	res *response.LoggedResponseWriter,
	req *http.Request,
) {
	// TODO: avoid err suppressing
	url, _ := url.Parse(target)
	fixRequest(*url, forwarding, req)

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ServeHTTP(res, req)

	// reqHeaders := utils.GetHeaders(req.Header)

	stored, err := storage.StoreGeneratedPage(req.Method, req.URL.String(), req.Header, *res)
	if !stored || err != nil {
		logger.Log(req, fmt.Sprintf("Not Stored: %v", err))
	}
}

func HandleRequest(lwr http.ResponseWriter, req *http.Request) {
	// TODO: COVERAGE
	if config.Config.Server.Forwarding.HTTP2HTTPS {
		RedirectToHTTPS(lwr, req, config.Config.Server.Forwarding.RedirectStatusCode)
		return
	}

	if req.Method == "PURGE" {
		HandlePurge(lwr, req)
		return
	}

	if req.Method == http.MethodConnect {
		lwr.WriteHeader(http.StatusMethodNotAllowed)
		// HandleTunneling(lwr, req)
	} else {
		HandleRequestAndProxy(lwr, req)
	}
}

// HandleRequestAndProxy - Handles the requests and proxies to backend server.
func HandleRequestAndProxy(res http.ResponseWriter, req *http.Request) {
	forwarding := config.GetForwarding()

	scheme := utils.IfEmpty(forwarding.Scheme, req.URL.Scheme)
	proxyURL := fmt.Sprintf("%s://%s", scheme, forwarding.Host)
	fullURL := proxyURL + req.URL.String()

	// reqHeaders := utils.GetHeaders(req.Header)

	lwr := response.NewLoggedResponseWriter(res)
	cached := storage.ServeCachedContent(lwr, req.Method, req.Header, fullURL)
	if !cached {
		serveReverseProxy(forwarding, proxyURL, lwr, req)
	}

	logger.LogRequest(req, lwr, cached)

}
