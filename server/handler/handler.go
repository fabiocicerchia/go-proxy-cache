package handler

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
	"fmt"
	"net/http"
	"time"

	"github.com/rs/xid"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/cache"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
)

// HttpMethodPurge - PURGE method.
const HttpMethodPurge = "PURGE"

// HandleRequest - Handles the entrypoint and directs the traffic to the right handler.
func HandleRequest(res http.ResponseWriter, req *http.Request) {
	tracingSpan, ctx := tracing.StartSpanFromRequest("server.handle_request", req)
	defer tracingSpan.End()

	telemetry.From(ctx).RegisterRequest(*req)

	rc, err := initRequestParams(ctx, res, req)
	if err != nil {
		tracing.AddErrorToSpan(tracingSpan, err)
		tracing.Fail(tracingSpan, "internal error")

		rc.GetLogger().Error(err.Error())
		return
	}

	telemetry.From(ctx).RegisterRequestCall(rc.ReqID, rc.Request, rc.GetRequestURL(), rc.GetScheme(), rc.IsWebSocket())

	rc.SetHSTSHeader()

	if rc.Request.Method == http.MethodConnect {
		if enableLoggingRequest {
			logger.LogRequest(rc.Request, rc.Response.StatusCode, rc.Response.Content.Len(), rc.ReqID, cache.StatusNA)
		}

		rc.SendMethodNotAllowed(ctx)

		return
	}

	if rc.GetScheme() == SchemeHTTP && rc.DomainConfig.Server.Upstream.HTTP2HTTPS {
		rc.RedirectToHTTPS(ctx)
		return
	}

	if rc.Route != nil {
		if redirect := redirectFilter(rc.Route); redirect != nil {
			rc.HandleRouteRedirect(ctx, redirect)
			return
		}
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

// NewRequestCall - Initialize a RequestCall object starting from incoming Request.
func NewRequestCall(res http.ResponseWriter, req *http.Request) RequestCall {
	reqID := xid.New().String()
	return RequestCall{
		ReqID:       reqID,
		RequestTime: time.Now(),
		Response:    response.NewLoggedResponseWriter(res, reqID),
		Request:     *req,
	}
}

func initRequestParams(ctx context.Context, res http.ResponseWriter, req *http.Request) (RequestCall, error) {
	var configFound bool

	rc := NewRequestCall(res, req)

	listeningPort := getListeningPort(req.Context())

	// Routed (Kubernetes ingress controller) mode: the routing table resolves
	// host AND path, which the static per-domain configuration cannot express.
	// An unmatched request is a 404, the ingress convention, rather than the
	// 501 the static path returns for an unknown virtual host.
	if router.Enabled() {
		route, found := router.Current().Match(req)
		if !found {
			rc.SendNotFound(ctx)

			logger.LogRequest(rc.Request, rc.Response.StatusCode, rc.Response.Content.Len(), rc.ReqID, cache.StatusMiss)

			return RequestCall{}, fmt.Errorf("No route matches %s%s.", rc.Request.Host, rc.Request.URL.Path)
		}

		rc.Route = route
		rc.DomainConfig = route.Config
	} else {
		rc.DomainConfig, configFound = config.DomainConf(req.Host, rc.GetScheme())
	}

	if (rc.Route == nil && !configFound) || !rc.IsLegitRequest(ctx, listeningPort) {
		rc.SendNotImplemented(ctx)

		logger.LogRequest(rc.Request, rc.Response.StatusCode, rc.Response.Content.Len(), rc.ReqID, cache.StatusMiss)

		return RequestCall{}, fmt.Errorf("Request for %s (listening on :%s) is not allowed (mostly likely it's a configuration mismatch).", rc.Request.Host, listeningPort)
	}

	return rc, nil
}
