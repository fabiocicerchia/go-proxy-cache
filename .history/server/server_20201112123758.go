package server

import (
	"log"
	"net/http"

	cache_redis "github.com/fabiocicerchia/go-proxy-cache/cache/redis"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// --- LOG

// Log the redirect url
func logRequest(proxyUrl string) {
	log.Printf("proxy_url: %s\n", proxyUrl)
}

// Log the env variables required for a reverse proxy
func logSetup() {
	forward_to := utils.GetEnv("FORWARD_TO", "")

	log.Printf("Server will run on: %s\n", getListenAddress())
	log.Printf("Redirecting to url: %s\n", forward_to)
}

// --- LOGIC

// Get the port to listen on
func getListenAddress() string {
	port := utils.GetEnv("PORT", "8080")
	return ":" + port
}

func Start() {
	// Log setup values
	logSetup()

	// redis connect
	cache_redis.Connect()

	// start server
	http.HandleFunc("/", handleRequestAndRedirect)

	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}
