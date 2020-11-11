package main

import (
	"github.com/fabiocicerchia/go-proxy-cache/log"
	"github.com/fabiocicerchia/go-proxy-cache/server"
)

func main() {
	// Log setup values
	log.LogSetup()

	// start server
	server.Start()
}
