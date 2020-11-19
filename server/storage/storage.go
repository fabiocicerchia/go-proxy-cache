package storage

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// CacheStatusHeader - HTTP Header for showing cache status
const CacheStatusHeader = "X-Go-Proxy-Cache-Status"

// CacheStatusHeaderHit - Cache status HIT for HTTP Header X-Go-Proxy-Cache-Status
const CacheStatusHeaderHit = "HIT"

// CacheStatusHeaderMiss - Cache status MISS for HTTP Header X-Go-Proxy-Cache-Status
const CacheStatusHeaderMiss = "MISS"

// ServeCachedContent - Retrives and sends to the client the cached response.
func ServeCachedContent(
	rw *response.LoggedResponseWriter,
	method string,
	reqHeaders http.Header,
	url string,
) bool {
	code, headers, page, err := cache.RetrieveFullPage(method, url, reqHeaders)
	if err != nil {
		log.Infof("Cannot retrieve page %s: %s\n", url, err)
	}

	if !cache.IsStatusAllowed(code) || len(page) == 0 {
		rw.Header().Set(CacheStatusHeader, CacheStatusHeaderMiss)

		return false
	}

	response.CopyHeaders(rw, headers)
	rw.Header().Set(CacheStatusHeader, CacheStatusHeaderHit)

	rw.WriteHeader(code)

	response.Flush(rw)

	return response.WriteBody(rw, page)
}

// StoreGeneratedPage - Stores a response in the cache.
func StoreGeneratedPage(
	method string,
	url string,
	reqHeaders http.Header,
	lwr response.LoggedResponseWriter,
) (bool, error) {
	status := lwr.StatusCode

	// headers := utils.GetHeaders(lwr.Header())

	content := string(lwr.Content)
	ttl := utils.GetTTL(lwr.Header(), config.Config.Cache.TTL)

	// TODO: pass obj
	done, err := cache.StoreFullPage(url, method, status, lwr.Header(), reqHeaders, content, ttl)

	return done, err
}
