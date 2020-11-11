package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res *loggedResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)

	storeGeneratedPage(req.URL.String(), *res)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	proxyURL := utils.GetProxyUrl()

	logRequest(proxyURL, req)

	fullURL := proxyURL + req.URL.String()
	if !serveCachedContent(res, fullURL) {
		lrw := newLoggedResponseWriter(res)
		serveReverseProxy(proxyURL, lrw, req)
	}
}
