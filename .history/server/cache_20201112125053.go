package server

import (
	"net/http"
	"strings"

	cache_redis "github.com/fabiocicerchia/go-proxy-cache/cache"
)

func serveCachedContent(res http.ResponseWriter, url string) bool {
	code, headers, page, _ := cache_redis.RetrieveFullPage(url)

	if code == http.StatusOK && page != "" {
		res.WriteHeader(code)
		for k, v := range headers {
			res.Header().Add(k, v)
		}
		res.Write(page)

		return true
	}

	return false
}

func storeGeneratedPage(url string, lrw loggedResponseWriter) {
	status := lrw.StatusCode
	headers := make(map[string]string)
	for k, values := range lrw.Header() {
		headers[k] = strings.Join(values, "")
	}
	content := string(lrw.Content)
	cache_redis.StoreFullPage(url, status, headers, content, 0)

}
