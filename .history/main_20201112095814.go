package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

/*
	Structs
*/

type requestPayloadStruct struct {
}

/*
	Utilities
*/

// Get env var or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

/*
	Getters
*/

// Get the port to listen on
func getListenAddress() string {
	port := getEnv("PORT", "1338")
	return ":" + port
}

// Get the url for a given proxy condition
func getProxyUrl() string {
	forward_to := os.Getenv("FORWARD_TO")

	return forward_to
}

/*
	Logging
*/

// Log the typeform payload and redirect url
func logRequestPayload(proxyUrl string) {
	log.Printf("proxy_condition: %s, proxy_url: %s\n", requestionPayload.ProxyCondition, proxyUrl)
}

// Log the env variables required for a reverse proxy
func logSetup() {
	forward_to := os.Getenv("FORWARD_TO")

	log.Printf("Server will run on: %s\n", getListenAddress())
	log.Printf("Redirecting to url: %s\n", forward_to)
}

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
	url := getProxyUrl()

	logRequestPayload(url)

	serveReverseProxy(url, res, req)
}

/*
	Entry
*/

func main() {
	// Log setup values
	logSetup()

	// start server
	http.HandleFunc("/", handleRequestAndRedirect)
	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}
