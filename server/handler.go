package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/roundrobin"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

func castToString(i interface{}) string {
	arr := i.([]string)
	if len(arr) > 0 {
		return arr[0]
	}

	return ""
}

func GetLBRoundRobin(endpoints []string, defaultHost string) string {
	lb := roundrobin.New([]interface{}{endpoints})
	endpoint, err := lb.Pick()
	if err != nil || castToString(endpoint) == "" {
		return defaultHost
	}

	return castToString(endpoint)
}

// Serve a reverse proxy for a given url
func serveReverseProxy(forwarding config.Forward, target string, res *LoggedResponseWriter, req *http.Request) {
	// parse the url
	// TODO: avoid err suppressing
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	scheme := utils.IfEmpty(forwarding.Scheme, url.Scheme) // TODO: TEST!!!
	host := utils.IfEmpty(forwarding.Host, url.Host)       // TODO: TEST!!!

	// Update the headers to allow for SSL redirection
	req.URL.Host = GetLBRoundRobin(config.Config.Server.Forwarding.Endpoints, url.Host)
	req.URL.Scheme = scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = host

	reqHeaders := utils.GetHeaders(req.Header)

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)

	done := storeGeneratedPage(req.URL.String(), reqHeaders, *res)
	LogRequest(req, res, done)
}

// Given a request send it to the appropriate url
func HandleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	forwarding := config.GetForwarding()

	proxyURL := fmt.Sprintf("%s://%s", forwarding.Scheme, forwarding.Host)

	reqHeaders := utils.GetHeaders(req.Header)

	fullURL := proxyURL + req.URL.String()
	if !serveCachedContent(res, reqHeaders, fullURL) {
		lrw := NewLoggedResponseWriter(res)
		serveReverseProxy(forwarding, proxyURL, lrw, req)
	}
}
