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

// RetrieveCachedContent - Retrives the cached response.
func RetrieveCachedContent(
	lwr *response.LoggedResponseWriter,
	req http.Request,
) (cache.URIObj, error) {
	method := req.Method
	reqHeaders := req.Header

	url := *req.URL
	url.Host = req.Host

	// TODO: duplication
	c := cache.CacheObj{
		AllowedStatuses: config.Config.Cache.AllowedStatuses,
		AllowedMethods:  config.Config.Cache.AllowedMethods,
	}

	err := c.RetrieveFullPage(method, url, reqHeaders)
	if err != nil {
		log.Warnf("Cannot retrieve page %s: %s\n", url.String(), err)
	}

	ok, err := c.IsValid()
	if !ok || err != nil {
		return cache.URIObj{}, err
	}

	return c.CurrentObj, nil
}

// StoreGeneratedPage - Stores a response in the cache.
func StoreGeneratedPage(
	req http.Request,
	lwr response.LoggedResponseWriter,
	domainConfigCache config.Cache,
) (bool, error) {
	ttl := ttl.GetTTL(lwr.Header(), domainConfigCache.TTL)

	// TODO: duplication
	c := cache.CacheObj{
		// TODO: convert to use domainConfigCache
		AllowedStatuses: config.Config.Cache.AllowedStatuses,
		AllowedMethods:  config.Config.Cache.AllowedMethods,
		CurrentObj: cache.URIObj{
			URL:             *req.URL,
			Host:            req.Host,
			Method:          req.Method,
			StatusCode:      lwr.StatusCode,
			RequestHeaders:  req.Header,
			ResponseHeaders: lwr.Header(),
			Content:         lwr.Content,
		},
	}

	done, err := c.StoreFullPage(ttl)

	return done, err
}

// PurgeCachedContent - Purges a content in the cache.
func PurgeCachedContent(scheme string, host string, req http.Request) (bool, error) {
	proxyURL := *req.URL
	proxyURL.Scheme = scheme
	proxyURL.Host = host

	// TODO: duplication
	c := cache.CacheObj{
		AllowedStatuses: config.Config.Cache.AllowedStatuses,
		AllowedMethods:  config.Config.Cache.AllowedMethods,
	}

	return c.PurgeFullPage(req.Method, proxyURL)
}
