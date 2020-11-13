package server

import (
	"net/http"

	cache_redis "github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
)

func Start() {
	// Init configs
	config.InitConfig()

	// Log setup values
	LogSetup(config.Config.Server.Port)

	// redis connect
	cache_redis.Connect(config.Config.Cache)

	// start server
	http.HandleFunc("/", handleRequestAndRedirect)

	port := ":" + config.Config.Server.Port
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}
