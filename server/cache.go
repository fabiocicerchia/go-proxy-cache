package server

import (
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	redis "github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

const CacheStatusHeader = "X-GoProxyCache-Status"
const CacheStatusHeaderHit = "HIT"
const CacheStatusHeaderMiss = "MISS"

func serveCachedContent(rw http.ResponseWriter, reqHeaders map[string]interface{}, url string) bool {
	code, headers, page, _ := redis.RetrieveFullPage(url, reqHeaders)

	if code == http.StatusOK && page != "" {
		CopyHeaders(rw, headers)
		rw.Header().Set(CacheStatusHeader, CacheStatusHeaderHit)

		rw.WriteHeader(code)

		Flush(rw)

		return WriteBody(rw, page)
	}

	rw.Header().Add(CacheStatusHeader, CacheStatusHeaderMiss)

	return false
}

func GetByKeyCaseInsensitive(items map[string]interface{}, key string) interface{} {
	keyLower := strings.ToLower(key)
	for k, v := range items {
		if strings.ToLower(k) == keyLower {
			return v
		}
	}

	return nil
}

func GetTTL(headers map[string]interface{}) time.Duration {
	ttl := time.Duration(config.Config.Server.TTL) * time.Second

	cacheControl := GetByKeyCaseInsensitive(headers, "Cache-Control")

	if cacheControl != nil {
		// TODO: add coverage
		cacheControlValue := strings.ToLower(cacheControl.(string))

		if strings.Contains(cacheControlValue, "no-cache") || strings.Contains(cacheControlValue, "no-store") {
			ttl = 0
		}

		// TODO: check which priority
		if maxage := GetTTLFrom("max-age", cacheControlValue); maxage > 0 {
			ttl = maxage
		}

		if smaxage := GetTTLFrom("s-maxage", cacheControlValue); smaxage > 0 {
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

func storeGeneratedPage(url string, reqHeaders map[string]interface{}, lrw LoggedResponseWriter) bool {
	status := lrw.StatusCode

	headers := utils.GetHeaders(lrw.Header())

	content := string(lrw.Content)
	ttl := GetTTL(headers)

	// TODO: pass obj
	done, err := redis.StoreFullPage(url, status, headers, reqHeaders, content, ttl)
	if err != nil {
		log.Printf("Error: %s\n", err)
	}

	return done
}
