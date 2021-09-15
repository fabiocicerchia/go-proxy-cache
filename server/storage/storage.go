package storage

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
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
func RetrieveCachedContent(rc RequestCallDTO, logger *log.Entry) (cache.URIObj, error) {
	err := rc.CacheObject.RetrieveFullPage()
	if err != nil {
		if err == cache.ErrEmptyValue {
			logger.Infof("Cannot retrieve page %s: %s\n", rc.CacheObject.CurrentURIObject.URL.String(), err) // TODO: Add to trace span?
		} else {
			logger.Warnf("Cannot retrieve page %s: %s\n", rc.CacheObject.CurrentURIObject.URL.String(), err) // TODO: Add to trace span?
		}

		return cache.URIObj{}, err
	}

	ok, err := rc.CacheObject.IsValid()
	if !ok || err != nil {
		return cache.URIObj{}, err
	}

	return rc.CacheObject.CurrentURIObject, nil
}

// StoreGeneratedPage - Stores a response in the cache.
func StoreGeneratedPage(rc RequestCallDTO, domainConfigCache config.Cache) (bool, error) {
	// Use the static rc.CacheObject.CurrentURIObject.ResponseHeaders to avoid data race
	currentTTL := ttl.GetTTL(rc.CacheObject.CurrentURIObject.ResponseHeaders, domainConfigCache.TTL)
	done, err := rc.CacheObject.StoreFullPage(currentTTL)

	return done, err
}

// PurgeCachedContent - Purges a content in the cache.
func PurgeCachedContent(upstream config.Upstream, rc RequestCallDTO) (bool, error) {
	return rc.CacheObject.PurgeFullPage()
}
