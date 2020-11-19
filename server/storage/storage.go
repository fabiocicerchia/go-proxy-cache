package storage

import (
	"net/http"
	"net/url"

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
	lwr *response.LoggedResponseWriter,
	req http.Request,
	url url.URL,
) bool {
	method := req.Method
	reqHeaders := req.Header

	code, headers, page, err := cache.RetrieveFullPage(method, url, reqHeaders)
	if err != nil {
		log.Infof("Cannot retrieve page %s: %s\n", url.String(), err)
	}

	if !cache.IsStatusAllowed(code) || len(page) == 0 {
		lwr.Header().Set(CacheStatusHeader, CacheStatusHeaderMiss)

		return false
	}

	response.CopyHeaders(lwr, headers)
	lwr.Header().Set(CacheStatusHeader, CacheStatusHeaderHit)

	lwr.WriteHeader(code)

	response.Flush(lwr)

	return response.WriteBody(lwr, page)
}

// StoreGeneratedPage - Stores a response in the cache.
func StoreGeneratedPage(
	req http.Request,
	lwr response.LoggedResponseWriter,
) (bool, error) {
	content := string(lwr.Content)
	ttl := utils.GetTTL(lwr.Header(), config.Config.Cache.TTL)

	response := cache.URIObj{
		URL:             *req.URL,
		Method:          req.Method,
		StatusCode:      lwr.StatusCode,
		RequestHeaders:  req.Header,
		ResponseHeaders: lwr.Header(),
		Content:         content,
	}

	done, err := cache.StoreFullPage(response, ttl)

	return done, err
}
