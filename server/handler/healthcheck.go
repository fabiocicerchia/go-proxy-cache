package handler

import (
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	log "github.com/sirupsen/logrus"
)

// HandleHealthcheck - Returns healthcheck status.
func HandleHealthcheck(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
	res.Write(([]byte)("HTTP OK\n"))

	if engine.Ping() {
		_, err := res.Write(([]byte)("REDIS OK\n"))
		if err != nil {
			log.Warnf("Error Writing: %s\n", err)
		}
	}
}
