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
	"fmt"
	"net/http"

	"github.com/rs/xid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/tracing"
)

// HttpMethodPurge - PURGE method.
const HttpMethodPurge = "PURGE"

// HandleRequest - Handles the entrypoint and directs the traffic to the right handler.
func HandleRequest(res http.ResponseWriter, req *http.Request) {
	ctx := otel.GetTextMapPropagator().Extract(req.Context(), propagation.HeaderCarrier(req.Header))

	ctx, tracingSpan := tracing.NewSpan(ctx, "server.handle_request")
	defer tracingSpan.End()
	tracing.AddTagsToSpan(tracingSpan, map[string]string{
		"request.host": req.Host,
		"request.url":  req.URL.String(),
	})

	rc, err := initRequestParams(res, req)
	if err != nil {
		tracing.AddErrorToSpan(tracingSpan, err)
		tracing.Fail(tracingSpan, "internal error")

		rc.GetLogger().Errorf(err.Error())
		return
	}

	reqURL := rc.GetRequestURL()
	tracing.AddTagsToSpan(tracingSpan, map[string]string{
		"request.id":        rc.ReqID,
		"request.full_url":  reqURL.String(),
		"request.method":    rc.Request.Method,
		"request.scheme":    rc.GetScheme(),
		"request.websocket": fmt.Sprintf("%v", rc.IsWebSocket()),
	})

	if rc.Request.Method == http.MethodConnect {
		if enableLoggingRequest {
			logger.LogRequest(rc.Request, *rc.Response, rc.ReqID, false, "-")
		}

		rc.SendMethodNotAllowed()

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

func initRequestParams(res http.ResponseWriter, req *http.Request) (RequestCall, error) {
	var configFound bool

	reqID := xid.New().String()
	rc := RequestCall{
		ReqID:       reqID,
		Response:    response.NewLoggedResponseWriter(res, reqID),
		Request:     *req,
		propagators: otel.GetTextMapPropagator(),
		tracer:      tracing.GetTracer(),
	}

	listeningPort := getListeningPort(req.Context())

	rc.DomainConfig, configFound = config.DomainConf(req.Host, rc.GetScheme())
	if !configFound || !rc.IsLegitRequest(listeningPort) {
		rc.SendNotImplemented()

		logger.LogRequest(rc.Request, *rc.Response, rc.ReqID, false, CacheStatusLabel[CacheStatusMiss])

		return RequestCall{}, fmt.Errorf("Request for %s (listening on :%s) is not allowed (mostly likely it's a configuration mismatch).", rc.Request.Host, listeningPort)
	}

	return rc, nil
}
