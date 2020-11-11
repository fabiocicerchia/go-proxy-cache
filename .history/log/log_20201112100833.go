package log

import (
	"log"
)

// Log the redirect url
func LogRequest(proxyUrl string) {
	log.Printf("proxy_url: %s\n", proxyUrl)
}

// Log the env variables required for a reverse proxy
func LogSetup() {
	forward_to := utils.GetEnv("FORWARD_TO")

	log.Printf("Server will run on: %s\n", server.GetListenAddress())
	log.Printf("Redirecting to url: %s\n", forward_to)
}
