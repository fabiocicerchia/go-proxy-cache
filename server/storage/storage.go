package storage

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/utils/ttl"
)

// RequestCallDTO - DTO object containing request and response.
type RequestCallDTO struct {
	ReqID       string
	Response    response.LoggedResponseWriter
	Request     http.Request
	CacheObject cache.Object
}

// RetrieveCachedContent - Retrives the cached response.
func RetrieveCachedContent(ctx context.Context, rc RequestCallDTO, logger *log.Entry) (cache.URIObj, error) {
	err := rc.CacheObject.RetrieveFullPage()
	if err != nil {
		escapedURL := strings.Replace(rc.CacheObject.CurrentURIObject.URL.String(), "\n", "", -1)
		escapedURL = strings.Replace(escapedURL, "\r", "", -1)
		if err == cache.ErrEmptyValue {
			logger.Infof("Cannot retrieve page %s: %s\n", escapedURL, err)
		} else {
			logger.Warnf("Cannot retrieve page %s: %s\n", escapedURL, err)
		}

		telemetry.From(ctx).RegisterEventWithData("Cannot retrieve page", map[string]string{
			"url":   rc.CacheObject.CurrentURIObject.URL.String(),
			"error": err.Error(),
		})

		return cache.URIObj{}, err
	}

	ok, err := rc.CacheObject.IsValid()
	if !ok || err != nil {
		return cache.URIObj{}, err
	}

	return rc.CacheObject.CurrentURIObject, nil
}

// ApplyNegativeTTL - Overrides the TTL for a configured status (e.g. 404/502)
// with its own short TTL instead of inheriting origin cache headers, which
// error pages rarely set sanely. This shields the origin from repeated hits
// on a broken URL.
func ApplyNegativeTTL(status int, negativeTTL map[int]int, defaultTTL time.Duration) time.Duration {
	if secs, ok := negativeTTL[status]; ok {
		return time.Duration(secs) * time.Second
	}

	return defaultTTL
}

// StoreGeneratedPage - Stores a response in the cache.
func StoreGeneratedPage(ctx context.Context, rc RequestCallDTO, domainConfigCache config.Cache) (bool, error) {
	// Use the static rc.CacheObject.CurrentURIObject.ResponseHeaders to avoid data race
	currentTTL := ttl.GetTTL(rc.CacheObject.CurrentURIObject.ResponseHeaders, domainConfigCache.TTL)
	currentTTL = ApplyNegativeTTL(rc.CacheObject.CurrentURIObject.StatusCode, domainConfigCache.NegativeTTL, currentTTL)

	done, err := rc.CacheObject.StoreFullPage(ctx, currentTTL)

	return done, err
}

// PurgeCachedContent - Purges a content in the cache.
func PurgeCachedContent(ctx context.Context, upstream config.Upstream, rc RequestCallDTO) (bool, error) {
	return rc.CacheObject.PurgeFullPage(ctx)
}
