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
	"fmt"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/rs/xid"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/tracing"
)

// HttpMethodPurge - PURGE method.
const HttpMethodPurge = "PURGE"

// HandleRequest - Handles the entrypoint and directs the traffic to the right handler.
func HandleRequest(res http.ResponseWriter, req *http.Request) {
	tracingSpan := tracing.StartSpanFromRequest("server.handle_request", req)
	defer tracingSpan.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), tracingSpan)

	metrics.IncRequestHost(req.Host)

	tracingSpan.
		SetTag(tracing.TagRequestHost, req.Host).
		SetTag(tracing.TagRequestUrl, req.URL.String())

	rc, err := initRequestParams(ctx, res, req)
	if err != nil {
		tracing.AddErrorToSpan(tracingSpan, err)
		tracing.Fail(tracingSpan, "internal error")

		rc.GetLogger().Errorf(err.Error())
		return
	}

	tracingSpan.SetBaggageItem(tracing.BaggageRequestID, rc.ReqID)

	reqURL := rc.GetRequestURL()
	tracingSpan.
		SetTag(tracing.TagRequestId, rc.ReqID).
		SetTag(tracing.TagRequestFullUrl, reqURL.String()).
		SetTag(tracing.TagRequestMethod, rc.Request.Method).
		SetTag(tracing.TagRequestScheme, rc.GetScheme()).
		SetTag(tracing.TagRequestWebsocket, rc.IsWebSocket())

	metrics.IncHttpMethod(rc.Request.Method)
	metrics.IncUrlScheme(rc.GetScheme())
	if rc.Request.Method == http.MethodConnect {
		if enableLoggingRequest {
			logger.LogRequest(rc.Request, *rc.Response, rc.ReqID, false, "-")
		}

		rc.SendMethodNotAllowed(ctx)

		return
	}

	if rc.GetScheme() == SchemeHTTP && rc.DomainConfig.Server.Upstream.HTTP2HTTPS {
		rc.RedirectToHTTPS(ctx)
		return
	}

	if rc.Request.Method == HttpMethodPurge {
		rc.HandlePurge(ctx)
		return
	}

	if rc.IsWebSocket() {
		rc.HandleWSRequestAndProxy(ctx)
	} else {
		rc.HandleHTTPRequestAndProxy(ctx)
	}
}

func NewRequestCall(res http.ResponseWriter, req *http.Request) RequestCall {
	reqID := xid.New().String()
	return RequestCall{
		ReqID:    reqID,
		Response: response.NewLoggedResponseWriter(res, reqID),
		Request:  *req,
	}
}

func initRequestParams(ctx context.Context, res http.ResponseWriter, req *http.Request) (RequestCall, error) {
	var configFound bool

	rc := NewRequestCall(res, req)

	listeningPort := getListeningPort(req.Context())

	rc.DomainConfig, configFound = config.DomainConf(req.Host, rc.GetScheme())
	if !configFound || !rc.IsLegitRequest(ctx, listeningPort) {
		rc.SendNotImplemented(ctx)

		logger.LogRequest(rc.Request, *rc.Response, rc.ReqID, false, CacheStatusLabel[CacheStatusMiss])

		return RequestCall{}, fmt.Errorf("Request for %s (listening on :%s) is not allowed (mostly likely it's a configuration mismatch).", rc.Request.Host, listeningPort)
	}

	return rc, nil
}
