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
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
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
		escapedURL := utils.EscapeLogValue(rc.CacheObject.CurrentURIObject.URL.String())
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

// StorageTTL - Decides the TTL a response is stored with, by precedence:
//  1. negative_ttl, for the statuses it lists;
//  2. override_ttl, when > 0, replacing any origin freshness information;
//  3. s-maxage / max-age / Expires from the origin (see ttl.GetTTL);
//  4. ttl, the default when the origin sends no freshness information.
//
// It never makes a response storable: StoreFullPage still refuses what a
// shared cache must not store (no-store, private, no-cache, Authorization
// without public/s-maxage/must-revalidate) whatever TTL is returned here.
func StorageTTL(uri cache.URIObj, domainConfigCache config.Cache) time.Duration {
	currentTTL := ttl.GetTTL(uri.ResponseHeaders, domainConfigCache.TTL)
	if domainConfigCache.OverrideTTL > 0 {
		currentTTL = time.Duration(domainConfigCache.OverrideTTL) * time.Second
	}

	return ApplyNegativeTTL(uri.StatusCode, domainConfigCache.NegativeTTL, currentTTL)
}

// StoreGeneratedPage - Stores a response in the cache.
func StoreGeneratedPage(ctx context.Context, rc RequestCallDTO, domainConfigCache config.Cache) (bool, error) {
	// Use the static rc.CacheObject.CurrentURIObject.ResponseHeaders to avoid data race
	currentTTL := StorageTTL(rc.CacheObject.CurrentURIObject, domainConfigCache)

	return rc.CacheObject.StoreFullPage(ctx, currentTTL)
}

// PurgeCachedContent - Purges a content in the cache, or everything cached for
// the host when the request-target is cache.PurgeAllPath.
func PurgeCachedContent(ctx context.Context, upstream config.Upstream, rc RequestCallDTO) (bool, error) {
	u := rc.CacheObject.CurrentURIObject.URL
	if u.Path == cache.PurgeAllPath && u.RawQuery == "" {
		return rc.CacheObject.PurgeHost(ctx)
	}

	return rc.CacheObject.PurgeFullPage(ctx)
}
