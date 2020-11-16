package handler

import (
	"fmt"
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
)

func HandlePurge(res http.ResponseWriter, req *http.Request) {
	forwarding := config.GetForwarding()

	proxyURL := fmt.Sprintf("%s://%s", forwarding.Scheme, forwarding.Host)
	fullURL := proxyURL + req.URL.String()

	cache.PurgeFullPage(req.Method, fullURL)

	res.WriteHeader(http.StatusOK)
	res.Write(([]byte)("OK"))
}
