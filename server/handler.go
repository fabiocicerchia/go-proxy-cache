package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/roundrobin"
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
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	scheme := url.Scheme
	if forwarding.Scheme != "" {
		scheme = forwarding.Scheme // TODO: TEST!!!
	}

	host := url.Host
	if forwarding.Host != "" {
		host = forwarding.Host // TODO: TEST!!!
	}

	// Update the headers to allow for SSL redirection
	req.URL.Host = GetLBRoundRobin(config.Config.Server.Forwarding.Endpoints, url.Host)
	req.URL.Scheme = scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = host

	// TODO: used twice -> method
	headersConverted := make(map[string]string)
	for k, v := range req.Header {
		headersConverted[k] = strings.Join(v, " ") // TODO: is correct join " " ?
	}

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)

	done := storeGeneratedPage(req.URL.String(), headersConverted, *res)
	LogRequest(req, res, done)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	forwarding := config.GetForwarding()
	proxyURL := forwarding.Scheme + "://" + forwarding.Host

	// TODO: used twice -> method
	headersConverted := make(map[string]string)
	for k, v := range req.Header {
		headersConverted[k] = strings.Join(v, " ") // TODO: is correct join " " ?
	}

	fullURL := proxyURL + req.URL.String()
	if !serveCachedContent(res, headersConverted, fullURL) {
		lrw := NewLoggedResponseWriter(res)
		serveReverseProxy(forwarding, proxyURL, lrw, req)
	}
}
