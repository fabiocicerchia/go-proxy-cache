package handler

import (
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
)

func HandleHealthcheck(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
	res.Write(([]byte)("HTTP OK\n"))

	if engine.Ping() {
		res.Write(([]byte)("REDIS OK\n"))
	}
}
