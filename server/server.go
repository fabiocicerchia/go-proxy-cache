package server

import (
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
)

// Start the GoProxyCache server.
func Start() {
	// Init configs
	config.InitConfigFromFileOrEnv("config.yml")

	// Log setup values
	logger.LogSetup(config.Config.Server.Forwarding, config.Config.Server.Port)

	// redis connect
	engine.Connect(config.Config.Cache)

	// start server
	http.HandleFunc("/healthcheck", handler.HandleHealthcheck)
	http.HandleFunc("/", handler.HandleRequestAndProxy)

	port := ":" + config.Config.Server.Port
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}
