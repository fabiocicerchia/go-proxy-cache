package server

import (
	"net/http"

	redis "github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
)

func Start() {
	// Init configs
	config.InitConfigFromFileOrEnv("config.yml")

	// Log setup values
	LogSetup(config.Config.Server.Forwarding, config.Config.Server.Port)

	// redis connect
	redis.Connect(config.Config.Cache)

	// start server
	http.HandleFunc("/", HandleRequestAndRedirect)

	port := ":" + config.Config.Server.Port
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}
