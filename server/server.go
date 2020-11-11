package server

import (
	"net/http"

	cache_redis "github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// Get the port to listen on
func getListenAddress() string {
	port := utils.GetEnv("SERVER_PORT", "8080")
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
