package handler

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"net/http"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/cache"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
)

// HandlePurge - Purges the cache for the requested URI.
func (rc RequestCall) HandlePurge(ctx context.Context) {
	rcDTO := ConvertToRequestCallDTO(rc)

	status, err := storage.PurgeCachedContent(ctx, rc.DomainConfig.Server.Upstream, rcDTO)
	if !status || err != nil {
		rc.Response.ForceWriteHeader(http.StatusNotFound)
		_ = rc.Response.WriteBody("KO")

		escapedURL := strings.Replace(rc.Request.URL.String(), "\n", "", -1)
		escapedURL = strings.Replace(escapedURL, "\r", "", -1)

		rc.GetLogger().Warnf("URL Not Purged %s: %v\n", escapedURL, err)

		telemetry.From(ctx).RegisterPurge(status, err)
		telemetry.From(ctx).RegisterStatusCode(http.StatusNotFound)

		return
	}

	rc.Response.ForceWriteHeader(http.StatusOK)
	_ = rc.Response.WriteBody("OK")

	telemetry.From(ctx).RegisterPurge(status, nil)
	telemetry.From(ctx).RegisterStatusCode(http.StatusOK)

	if enableLoggingRequest {
		logger.LogRequest(rc.Request, rc.Response.StatusCode, rc.Response.Content.Len(), rc.ReqID, cache.StatusNA)
	}
}
