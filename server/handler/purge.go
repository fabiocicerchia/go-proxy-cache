package handler

import (
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	log "github.com/sirupsen/logrus"
)

// HandlePurge - Purges the cache for the requested URI.
func HandlePurge(lwr *response.LoggedResponseWriter, req *http.Request) {
	forwarding := config.GetForwarding()

	scheme := utils.IfEmpty(forwarding.Scheme, req.URL.Scheme)

	proxyURL := *req.URL
	proxyURL.Scheme = scheme
	proxyURL.Host = forwarding.Host

	status, err := cache.PurgeFullPage(req.Method, proxyURL)

	if !status || err != nil {
		lwr.WriteHeader(http.StatusNotFound)
		_ = response.WriteBody(lwr, "KO")

		log.Warnf("URL Not Purged %s: %v\n", proxyURL.String(), err)
		return
	}

	lwr.WriteHeader(http.StatusOK)
	_ = response.WriteBody(lwr, "OK")
}
