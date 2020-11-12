package server

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	cache_redis "github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
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

func getTTL(headers map[string]interface{}) time.Duration {
	ttlSecs, _ := strconv.Atoi(utils.GetEnv("DEFAULT_TTL", "0"))
	ttl := time.Duration(ttlSecs) * time.Second

	if _, ok := headers["Cache-Control"]; ok {
		cacheControl := headers["Cache-Control"].(string)

		if maxage := getTTLFrom("max-age", cacheControl); maxage > 0 {
			ttl = maxage
		}

		if smaxage := getTTLFrom("s-maxage", cacheControl); smaxage > 0 {
			ttl = smaxage
		}
	}

	return ttl
}

func getTTLFrom(cacheType string, cacheControl string) time.Duration {
	var ttl time.Duration

	ageRegex := regexp.MustCompile(`max-age=(?P<TTL>\d+)`)
	age := ageRegex.FindStringSubmatch(cacheControl)

	if len(age) > 0 {
		ageTTL, _ := strconv.ParseInt(age[0], 10, 64)
		ttl = time.Duration(ageTTL)
	}

	return ttl
}

func storeGeneratedPage(url string, lrw loggedResponseWriter) {
	status := lrw.StatusCode

	headers := make(map[string]interface{})
	for k, values := range lrw.Header() {
		headers[k] = strings.Join(values, "")
	}

	content := string(lrw.Content)
	ttl := getTTL(headers)

	cache_redis.StoreFullPage(url, status, headers, content, ttl)
}
