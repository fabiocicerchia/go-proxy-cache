package utils

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GetTTL - Retrieves TTL is seconds from Expires and Cache-Control HTTP headers.
func GetTTL(headers map[string]interface{}, defaultTTL int) time.Duration {
	ttl := time.Duration(defaultTTL) * time.Second

	expires := GetByKeyCaseInsensitive(headers, "Expires")

	if expires != nil {
		expiresValue := expires.(string)

		expiresDate, err := http.ParseTime(expiresValue)
		if err == nil {
			diff := expiresDate.UTC().Sub(time.Now().UTC())
			if diff > 0 {
				ttl = time.Duration(diff)
			}
		}
	}

	cacheControl := GetByKeyCaseInsensitive(headers, "Cache-Control")

	if cacheControl != nil {
		cacheControlValue := strings.ToLower(cacheControl.(string))

		// TODO: check which priority
		if maxage := GetTTLFromCacheControl("max-age", cacheControlValue); maxage > 0 {
			ttl = maxage
		}

		if smaxage := GetTTLFromCacheControl("s-maxage", cacheControlValue); smaxage > 0 {
			ttl = smaxage
		}

		if strings.Contains(cacheControlValue, "no-cache") || strings.Contains(cacheControlValue, "no-store") {
			ttl = 0
		}
	}

	return ttl
}

// GetTTLFromCacheControl - Retrieves TTL value from Cache-Control header.
func GetTTLFromCacheControl(cacheType string, cacheControl string) time.Duration {
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
