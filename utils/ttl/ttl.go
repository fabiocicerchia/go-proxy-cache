package ttl

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

func ttlFromExpires(expiresValue string) *time.Duration {
	expiresDate, err := http.ParseTime(expiresValue)
	if err == nil {
		diff := expiresDate.UTC().Sub(time.Now().UTC())
		if diff > 0 {
			return &diff
		}
	}

	return nil
}

func ttlFromCacheControlChain(cacheControlValue string) *time.Duration {
	// Ref: https://tools.ietf.org/html/rfc7234#section-4.2.1
	if strings.Contains(cacheControlValue, "no-cache") || strings.Contains(cacheControlValue, "no-store") {
		zeroDuration := time.Duration(0)
		return &zeroDuration
	}

	if smaxage := GetTTLFromCacheControl("s-maxage", cacheControlValue); smaxage > 0 {
		return &smaxage
	}

	if maxage := GetTTLFromCacheControl("max-age", cacheControlValue); maxage > 0 {
		return &maxage
	}

	return nil
}

// GetTTL - Retrieves TTL is seconds from Expires and Cache-Control HTTP headers.
func GetTTL(headers http.Header, defaultTTL int) time.Duration {
	ttl := time.Duration(defaultTTL) * time.Second

	expires := slice.GetByKeyCaseInsensitive(headers, "Expires")
	ttl = overrideWithExpires(ttl, expires)

	cacheControl := slice.GetByKeyCaseInsensitive(headers, "Cache-Control")
	ttl = overrideWithCacheControl(ttl, cacheControl)

	return ttl
}

func overrideWithExpires(ttl time.Duration, expires interface{}) time.Duration {
	if expires != nil {
		expiresValue := expires.([]string)[0]
		expiresTTL := ttlFromExpires(expiresValue)

		if expiresTTL != nil {
			return *expiresTTL
		}
	}

	return ttl
}

func overrideWithCacheControl(ttl time.Duration, cacheControl interface{}) time.Duration {
	if cacheControl != nil {
		cacheControlValue := strings.ToLower(cacheControl.([]string)[0])
		cacheControlTTL := ttlFromCacheControlChain(cacheControlValue)

		if cacheControlTTL != nil {
			return *cacheControlTTL
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
		ageTTL, _ := strconv.ParseInt(age[1], 10, 64) // it fallbacks on 0
		ttl = time.Duration(ageTTL) * time.Second
	}

	return ttl
}
