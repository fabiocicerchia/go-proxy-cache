package ttl

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/http"
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

// ParseCacheControl - Parses every Cache-Control header field into a map of
// lowercased directive name to its (unquoted) argument, per RFC 9111 §5.2.
// Multiple fields are combined as one comma-separated list (RFC 9110 §5.3).
// When a directive is repeated, the first occurrence wins.
func ParseCacheControl(headers http.Header) map[string]string {
	directives := map[string]string{}

	for key, values := range headers {
		if !strings.EqualFold(key, "Cache-Control") {
			continue
		}

		for _, value := range values {
			for _, part := range splitDirectives(value) {
				name, arg, _ := strings.Cut(part, "=")
				name = strings.ToLower(strings.TrimSpace(name))
				if _, seen := directives[name]; name == "" || seen {
					continue
				}
				directives[name] = strings.Trim(strings.TrimSpace(arg), `"`)
			}
		}
	}

	return directives
}

// splitDirectives splits on commas that are not inside a quoted-string, so
// private="Set-Cookie, X-Foo" stays one directive.
func splitDirectives(value string) []string {
	var parts []string

	inQuotes := false
	start := 0

	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '"':
			inQuotes = !inQuotes
		case '\\':
			if inQuotes {
				i++
			}
		case ',':
			if !inQuotes {
				parts = append(parts, value[start:i])
				start = i + 1
			}
		}
	}

	return append(parts, value[start:])
}

// ForbidsStoring - Reports whether the response directives forbid a shared
// cache from storing the response:
//   - no-store (RFC 9111 §5.2.2.5);
//   - private, qualified or not (§5.2.2.7): a qualified private="Field" would
//     allow storing the response minus those fields, but a full-page cache
//     doesn't strip headers, so it's treated as unqualified private;
//   - no-cache (§5.2.2.4): the RFC allows storing it if always revalidated;
//     this cache doesn't revalidate, so it's never stored (stricter, safe).
func ForbidsStoring(directives map[string]string) bool {
	for _, name := range []string{"no-store", "private", "no-cache"} {
		if _, ok := directives[name]; ok {
			return true
		}
	}

	return false
}

// parseDeltaSeconds - An invalid or negative delta-seconds is treated as 0
// (stale), as RFC 9111 §1.2.2 and §4.2.1 advise.
func parseDeltaSeconds(value string) time.Duration {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds < 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

func ttlFromCacheControlChain(directives map[string]string) *time.Duration {
	if ForbidsStoring(directives) {
		zeroDuration := time.Duration(0)
		return &zeroDuration
	}

	// s-maxage overrides max-age for a shared cache (RFC 9111 §5.2.2.10).
	for _, name := range []string{"s-maxage", "max-age"} {
		if arg, ok := directives[name]; ok {
			ttl := parseDeltaSeconds(arg)
			return &ttl
		}
	}

	return nil
}

// GetTTL - Retrieves TTL is seconds from Expires and Cache-Control HTTP headers.
func GetTTL(headers http.Header, defaultTTL int) time.Duration {
	ttl := time.Duration(defaultTTL) * time.Second

	expires := slice.GetByKeyCaseInsensitive(headers, "Expires")
	ttl = overrideWithExpires(ttl, expires)

	if cacheControlTTL := ttlFromCacheControlChain(ParseCacheControl(headers)); cacheControlTTL != nil {
		ttl = *cacheControlTTL
	}

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

// GetTTLFromCacheControl - Retrieves TTL value from Cache-Control header.
func GetTTLFromCacheControl(cacheType string, cacheControl string) time.Duration {
	directives := ParseCacheControl(http.Header{"Cache-Control": []string{cacheControl}})

	return parseDeltaSeconds(directives[cacheType])
}
