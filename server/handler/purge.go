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
	"context"
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/server/tracing"
)

// HandlePurge - Purges the cache for the requested URI.
func (rc RequestCall) HandlePurge(ctx context.Context) {
	rcDTO := ConvertToRequestCallDTO(rc)

	status, err := storage.PurgeCachedContent(rc.DomainConfig.Server.Upstream, rcDTO)
	if !status || err != nil {
		rc.Response.ForceWriteHeader(http.StatusNotFound)
		_ = rc.Response.WriteBody("KO")

		rc.GetLogger().Warnf("URL Not Purged %s: %v\n", rc.Request.URL.String(), err)

		tracing.SpanFromContext(ctx).
			SetTag("purge.status", status).
			SetTag("response.status_code", http.StatusNotFound)

		if err != nil {
			tracing.AddErrorToSpan(tracing.SpanFromContext(ctx), err)
			tracing.Fail(tracing.SpanFromContext(rc.Request.Context()), "internal error")
		}

		return
	}

	rc.Response.ForceWriteHeader(http.StatusOK)
	_ = rc.Response.WriteBody("OK")

	tracing.SpanFromContext(ctx).
		SetTag("purge.status", status).
		SetTag("response.status_code", http.StatusOK)

	if enableLoggingRequest {
		logger.LogRequest(rc.Request, *rc.Response, rc.ReqID, false, "-")
	}
}
