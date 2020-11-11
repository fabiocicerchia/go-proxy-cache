package server

import (
	"net/http"
)

// Get the port to listen on
func GetListenAddress() string {
	port := utils.GetEnv("PORT", "8080")
	return ":" + port
}

func Start() {
	// start server
	http.HandleFunc("/", handleRequestAndRedirect)
	if err := http.ListenAndServe(server.GetListenAddress(), nil); err != nil {
		panic(err)
	}
}
