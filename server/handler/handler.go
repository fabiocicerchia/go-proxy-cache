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

	rc.DomainConfig, configFound = config.DomainConf(req.Host, rc.GetScheme())
	if !configFound || !rc.IsLegitRequest(ctx, listeningPort) {
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
