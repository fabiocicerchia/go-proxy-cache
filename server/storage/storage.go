package storage

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// RetrieveCachedContent - Retrives the cached response.
func RetrieveCachedContent(
	lwr *response.LoggedResponseWriter,
	req http.Request,
) (int, http.Header, [][]byte, error) {
	method := req.Method
	reqHeaders := req.Header

	url := *req.URL
	// url.Host = req.Host

	code, headers, content, err := cache.RetrieveFullPage(method, url, reqHeaders)
	if err != nil {
		log.Warnf("Cannot retrieve page %s: %s\n", url.String(), err)
	}

	if !cache.IsStatusAllowed(code) || utils.LenSliceBytes(content) == 0 {
		return 0, http.Header{}, [][]byte{}, fmt.Errorf("Not allowed. Status %d - Content length %d", code, len(content))
	}

	return code, headers, content, nil
}

// StoreGeneratedPage - Stores a response in the cache.
func StoreGeneratedPage(
	req http.Request,
	lwr response.LoggedResponseWriter,
) (bool, error) {
	ttl := utils.GetTTL(lwr.Header(), config.Config.Cache.TTL)

	response := cache.URIObj{
		URL:             *req.URL,
		Host:            req.Host,
		Method:          req.Method,
		StatusCode:      lwr.StatusCode,
		RequestHeaders:  req.Header,
		ResponseHeaders: lwr.Header(),
		Content:         lwr.Content,
		// ContentTwo:      lwr.ContentTwo,
	}

	done, err := cache.StoreFullPage(response, ttl)

	return done, err
}
