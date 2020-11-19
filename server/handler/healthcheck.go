package handler

import (
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

// HandleHealthcheck - Returns healthcheck status.
func HandleHealthcheck(res http.ResponseWriter, req *http.Request) {
	lwr := response.NewLoggedResponseWriter(res)

	lwr.WriteHeader(http.StatusOK)
	_ = response.WriteBody(lwr, "HTTP OK\n")

	if engine.Ping() {
		_ = response.WriteBody(lwr, "REDIS OK\n")
	} else {
		_ = response.WriteBody(lwr, "REDIS KO\n")
	}
}
