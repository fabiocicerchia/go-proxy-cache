package server

import (
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	cache_redis "github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
)

const CacheStatusHeader = "X-GoProxyCache-Status"
const CacheStatusHeaderHit = "HIT"
const CacheStatusHeaderMiss = "MISS"

func serveCachedContent(rw http.ResponseWriter, reqHeaders map[string]string, url string) bool {
	code, headers, page, _ := cache_redis.RetrieveFullPage(url, reqHeaders)

	if code == http.StatusOK && page != "" {
		for k, v := range headers {
			rw.Header().Set(k, v)
		}
		rw.Header().Set(CacheStatusHeader, CacheStatusHeaderHit)

		rw.WriteHeader(code)

		if fl, ok := rw.(http.Flusher); ok {
			fl.Flush()
		}

		pageByte := []byte(page)
		sent, err := rw.Write(pageByte)
		// try again
		if sent == 0 && err != nil {
			// TODO: LOG
			sent, err = rw.Write(pageByte)
			return sent > 0 && err == nil
		}

		return true
	}

	rw.Header().Add(CacheStatusHeader, CacheStatusHeaderMiss)

	return false
}

func GetTTL(headers map[string]interface{}) time.Duration {
	ttl := time.Duration(config.Config.Server.TTL) * time.Second

	if _, ok := headers["Cache-Control"]; ok {
		cacheControl := headers["Cache-Control"].(string)

		if maxage := GetTTLFrom("max-age", cacheControl); maxage > 0 {
			ttl = maxage
		}

		if smaxage := GetTTLFrom("s-maxage", cacheControl); smaxage > 0 {
			ttl = smaxage
		}
	}

	return ttl
}

func GetTTLFrom(cacheType string, cacheControl string) time.Duration {
	var ttl time.Duration
	ttl = 0 * time.Second

	ageRegex := regexp.MustCompile(cacheType + `=(?P<TTL>\d+)`)
	age := ageRegex.FindStringSubmatch(cacheControl)

	if len(age) > 0 {
		ageTTL, _ := strconv.ParseInt(age[1], 10, 64)
		ttl = time.Duration(ageTTL) * time.Second
	}

	return ttl
}

func storeGeneratedPage(url string, reqHeaders map[string]string, lrw LoggedResponseWriter) bool {
	status := lrw.StatusCode

	headers := make(map[string]interface{})
	for k, values := range lrw.Header() {
		headers[k] = strings.Join(values, "")
	}

	content := string(lrw.Content)
	ttl := GetTTL(headers)

	done, err := cache_redis.StoreFullPage(url, status, headers, reqHeaders, content, ttl)
	if err != nil {
		log.Printf("Error: %s\n", err)
	}

	return done
}
