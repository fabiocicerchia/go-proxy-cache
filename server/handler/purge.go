package handler

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

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	log "github.com/sirupsen/logrus"
)

// HandlePurge - Purges the cache for the requested URI.
func (rc RequestCall) HandlePurge(domainConfig *config.Configuration) {
	forwarding := domainConfig.Server.Forwarding
	scheme := utils.IfEmpty(forwarding.Scheme, rc.GetScheme())

	status, err := storage.PurgeCachedContent(scheme, forwarding.Host, *rc.Request)
	if !status || err != nil {
		rc.Response.WriteHeader(http.StatusNotFound)
		_ = rc.Response.WriteBody("KO")

		log.Warnf("URL Not Purged %s: %v\n", rc.Request.URL.String(), err)
		return
	}

	rc.Response.WriteHeader(http.StatusOK)
	_ = rc.Response.WriteBody("OK")
}
