package server

import (
	"net/http"
	"strings"

	cache_redis "github.com/fabiocicerchia/go-proxy-cache/cache"
)

const CacheStatusHeader = "X-GoProxyCache-Status"
const CacheStatusHeaderHit = "HIT"
const CacheStatusHeaderMiss = "MISS"

func serveCachedContent(rw http.ResponseWriter, url string) bool {
	code, headers, page, _ := cache_redis.RetrieveFullPage(url)

	if code == http.StatusOK && page != "" {
		rw.WriteHeader(code)

		for k, v := range headers {
			rw.Header().Set(k, v)
		}
		rw.Header().Set(CacheStatusHeader, CacheStatusHeaderHit)
		rw.Header().Write(rw)

		if fl, ok := rw.(http.Flusher); ok {
			fl.Flush()
		}

		pageByte := []byte(page)
		rw.Write(pageByte)

		return true
	}

	rw.Header().Add(CacheStatusHeader, CacheStatusHeaderMiss)

	return false
}

func storeGeneratedPage(url string, lrw loggedResponseWriter) {
	status := lrw.StatusCode
	headers := make(map[string]interface{})
	for k, values := range lrw.Header() {
		headers[k] = strings.Join(values, "")
	}
	content := string(lrw.Content)
	cache_redis.StoreFullPage(url, status, headers, content, 0)
}
