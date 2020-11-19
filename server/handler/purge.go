package handler

import (
	"fmt"
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	log "github.com/sirupsen/logrus"
)

// HandlePurge - Purges the cache for the requested URI.
func HandlePurge(res http.ResponseWriter, req *http.Request) {
	forwarding := config.GetForwarding()

	scheme := utils.IfEmpty(forwarding.Scheme, req.URL.Scheme)
	proxyURL := fmt.Sprintf("%s://%s", scheme, forwarding.Host)
	fullURL := proxyURL + req.URL.String()

	status, err := cache.PurgeFullPage(req.Method, fullURL)

	if !status || err != nil {
		res.WriteHeader(http.StatusNotFound)
		_ = response.WriteBody(res, "KO")

		log.Warnf("URL Not Purged %s: %v\n", fullURL, err)
		return
	}

	res.WriteHeader(http.StatusOK)
	_ = response.WriteBody(res, "OK")
}
