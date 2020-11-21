package handler

import (
	"bytes"
	"fmt"
	"io/ioutil"
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

func FixRequest(url url.URL, forwarding config.Forward, req *http.Request) {
	scheme := utils.IfEmpty(forwarding.Scheme, url.Scheme)
	host := utils.IfEmpty(forwarding.Host, url.Host)

	req.URL.Host = balancer.GetLBRoundRobin(url.Host)
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
	code, headers, chunks, err := storage.RetrieveCachedContent(lwr, req)
	if err != nil {
		lwr.Header().Set(response.CacheStatusHeader, response.CacheStatusHeaderMiss)

		log.Warnf("Error on serving cached content: %s", err)
		return false
	}

	var bodyBytes []byte
	for _, chunk := range chunks {
		bodyBytes = append(bodyBytes, chunk...)
	}
	body := ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	// body := ioutil.NopCloser(bytes.NewBuffer([]byte(chunks)))

	res := http.Response{
		StatusCode: code,
		Header:     headers,
		Body:       body,
	}
	res.Header.Set(response.CacheStatusHeader, response.CacheStatusHeaderHit)

	ctx := req.Context()
	transport.ServeResponse(ctx, lwr, res, url, chunks)

	return true
}

func HandleRequest(res http.ResponseWriter, req *http.Request) {
	lwr := response.NewLoggedResponseWriter(res)

	if req.URL.Scheme == "http" && config.Config.Server.Forwarding.HTTP2HTTPS {
		RedirectToHTTPS(lwr.ResponseWriter, req, config.Config.Server.Forwarding.RedirectStatusCode)
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
	forwarding := config.GetForwarding()

	scheme := utils.IfEmpty(forwarding.Scheme, req.URL.Scheme)

	proxyURL := *req.URL
	proxyURL.Scheme = scheme
	proxyURL.Host = forwarding.Host

	cached := serveCachedContent(lwr, *req, proxyURL)
	if !cached {
		serveReverseProxy(forwarding, proxyURL, lwr, req)
	}

	logger.LogRequest(*req, *lwr, cached)
}
