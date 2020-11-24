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
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// RetrieveCachedContent - Retrives the cached response.
func RetrieveCachedContent(
	lwr *response.LoggedResponseWriter,
	req http.Request,
) (cache.URIObj, error) {
	method := req.Method
	reqHeaders := req.Header

	url := *req.URL

	uriobj, err := cache.RetrieveFullPage(method, url, reqHeaders)
	if err != nil {
		log.Warnf("Cannot retrieve page %s: %s\n", url.String(), err)
	}

	if !cache.IsStatusAllowed(uriobj.StatusCode) || utils.LenSliceBytes(uriobj.Content) == 0 {
		return uriobj, fmt.Errorf(
			"Not allowed. Status %d - Content length %d",
			uriobj.StatusCode,
			utils.LenSliceBytes(uriobj.Content),
		)
	}

	return uriobj, nil
}

// StoreGeneratedPage - Stores a response in the cache.
func StoreGeneratedPage(
	req http.Request,
	lwr response.LoggedResponseWriter,
) (bool, error) {
	domainConfig := config.DomainConf(req.Host)
	ttl := utils.GetTTL(lwr.Header(), domainConfig.Cache.TTL)

	response := cache.URIObj{
		URL:             *req.URL,
		Host:            req.Host,
		Method:          req.Method,
		StatusCode:      lwr.StatusCode,
		RequestHeaders:  req.Header,
		ResponseHeaders: lwr.Header(),
		Content:         lwr.Content,
	}

	done, err := cache.StoreFullPage(response, ttl)

	return done, err
}
