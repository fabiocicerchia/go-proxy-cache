package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

/*
	Structs
*/

var ctx = context.Background()

/*
	Getters
*/

/*
	Reverse Proxy Logic
*/

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
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
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	url := utils.GetProxyUrl()

	log.LogRequest(url)
	serveReverseProxy(url, res, req)
}

/*
	Entry
*/

func main() {
	// Log setup values
	log.LogSetup()

	// start server
	server.Start()
}
