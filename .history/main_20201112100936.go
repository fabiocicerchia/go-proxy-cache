package main

import (
	"context"
	"log"
)

var ctx = context.Background()

func main() {
	// Log setup values
	log.LogSetup()

	// start server
	server.Start()
}
