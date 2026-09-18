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

	// In routed mode the matched route carries the authentication settings,
	// and only the route does: one host can be split across several objects
	// with different settings, so resolving by Host picks an arbitrary one.
	if rc.Route != nil && routeAuthorizer != nil {
		if err := routeAuthorizer(res, req, &rc.DomainConfig.Jwt); err != nil {
			logger.LogRequest(rc.Request, http.StatusUnauthorized, 0, rc.ReqID, cache.StatusMiss)

			return
		}
	}

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

// routeAuthorizer - Validates a request against the matched route's settings.
//
// Injected rather than imported: the authentication package already depends on
// this one. Nil on the static path, which authenticates in middleware.
var routeAuthorizer func(http.ResponseWriter, *http.Request, *config.Jwt) error

// SetRouteAuthorizer - Installs the per-route request authorizer.
func SetRouteAuthorizer(authorize func(http.ResponseWriter, *http.Request, *config.Jwt) error) {
	routeAuthorizer = authorize
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

	// The routing table resolves host and path together, which the static
	// per-domain configuration cannot express. An unmatched request is a 404
	// here, not the 501 the static path returns for an unknown virtual host.
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

		return RequestCall{}, fmt.Errorf("Request for %s (listening on :%s) is not allowed: %s", rc.Request.Host, listeningPort, rejectionReason(rc, configFound, listeningPort))
	}

	return rc, nil
}

// rejectionReason - Says which of the three checks above turned the request
// away, in the terms of the config file that decides it.
//
// "mostly likely it's a configuration mismatch" was true and unactionable: the
// two comparisons that produce it were only visible at debug level, so a 501 on
// every request looked like the proxy refusing to work rather than the proxy
// saying the Host header and the config disagree.
func rejectionReason(rc RequestCall, configFound bool, listeningPort string) string {
	if !configFound {
		return fmt.Sprintf("no domain in the config matches host %q with scheme %q", rc.GetHostname(), rc.GetScheme())
	}

	if rc.DomainConfig.Server.Upstream.Host != rc.GetHostname() {
		return fmt.Sprintf(
			"request Host %q does not match server.upstream.host %q for this domain -- the two have to be equal; to send that hostname to a different address, keep it as upstream.host and list the address in server.upstream.endpoints",
			rc.GetHostname(),
			rc.DomainConfig.Server.Upstream.Host,
		)
	}

	if !isLegitPort(rc.DomainConfig.Server.Port, listeningPort) {
		return fmt.Sprintf(
			"the port this request arrived on (:%s) is neither server.port.http (%q) nor server.port.https (%q) for this domain",
			listeningPort,
			rc.DomainConfig.Server.Port.HTTP,
			rc.DomainConfig.Server.Port.HTTPS,
		)
	}

	return "no reason recorded, which is a bug in this function rather than in the request"
}
