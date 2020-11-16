package storage

import (
	"log"
	"net/http"

	redis "github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

const CacheStatusHeader = "X-GoProxyCache-Status"
const CacheStatusHeaderHit = "HIT"
const CacheStatusHeaderMiss = "MISS"

func ServeCachedContent(rw http.ResponseWriter, method string, reqHeaders map[string]interface{}, url string) bool {
	code, headers, page, _ := redis.RetrieveFullPage(method, url, reqHeaders)

	if code != http.StatusOK || page != "" {
		rw.Header().Set(CacheStatusHeader, CacheStatusHeaderMiss)

		return false
	}

	response.CopyHeaders(rw, headers)
	rw.Header().Set(CacheStatusHeader, CacheStatusHeaderHit)

	rw.WriteHeader(code)

	response.Flush(rw)

	return response.WriteBody(rw, page)
}

func StoreGeneratedPage(method, url string, reqHeaders map[string]interface{}, lwr response.LoggedResponseWriter) bool {
	status := lwr.StatusCode

	headers := utils.GetHeaders(lwr.Header())

	content := string(lwr.Content)
	ttl := utils.GetTTL(headers, config.Config.Server.TTL)

	// TODO: pass obj
	done, err := redis.StoreFullPage(url, method, status, headers, reqHeaders, content, ttl)
	if err != nil {
		log.Printf("Error: %s\n", err)
	}

	return done
}
