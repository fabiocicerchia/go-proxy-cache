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
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/yhat/wsutil"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
)

// SchemeHTTPS - HTTPS scheme.
var SchemeHTTPS string = "https"

// SchemeHTTP - HTTP scheme.
var SchemeHTTP string = "http"

// SchemeWS - WS scheme.
var SchemeWS string = "ws"

// SchemeWSS - WSS scheme.
var SchemeWSS string = "wss"

// RequestCall - Main object containing request and response.
type RequestCall struct {
	ReqID        string
	Response     *response.LoggedResponseWriter
	Request      http.Request
	DomainConfig config.Configuration
	propagators  propagation.TextMapPropagator
	tracer       trace.Tracer
}

// GetLogger - Get logger instance with RequestID.
func (rc RequestCall) GetLogger() *log.Entry {
	return logger.GetGlobal().WithFields(log.Fields{
		"ReqID": rc.ReqID,
	})
}

// IsLegitRequest - Check whether a request is bound on the right Host and Port.
func (rc RequestCall) IsLegitRequest(ctx context.Context, listeningPort string) bool {
	hostMatch := rc.DomainConfig.Server.Upstream.Host == rc.GetHostname()
	legitPort := isLegitPort(rc.DomainConfig.Server.Port, listeningPort)

	telemetry.From(ctx).RegisterLegitRequest(hostMatch, legitPort, rc.GetHostname(), listeningPort, rc.DomainConfig.Server.Upstream.Host, rc.DomainConfig.Server.Upstream.Port)

	rc.GetLogger().Debugf("Is Hostname matching Request and Configuration? %v - Request: %s - Config: %s", hostMatch, rc.GetHostname(), rc.DomainConfig.Server.Upstream.Host)
	rc.GetLogger().Debugf("Is Port matching Request and Configuration? %v - Request: %s - Config: %s", legitPort, listeningPort, rc.DomainConfig.Server.Port)

	return hostMatch && legitPort
}

// GetRequestURL - Returns the valid Request URL (with Scheme and Host).
func (rc RequestCall) GetRequestURL() url.URL {
	// Original URL does not have scheme and hostname in it, so we need to add it.
	// Ref: https://github.com/golang/go/issues/28940
	url := *rc.Request.URL
	url.Scheme = rc.GetScheme()
	url.Host = rc.GetHostname()

	return url
}

// GetHostname - Returns only the hostname (without port if present).
func (rc RequestCall) GetHostname() string {
	return strings.Split(rc.Request.Host, ":")[0]
}

// GetScheme - Returns current request scheme.
// For server requests the URL is parsed from the URI supplied on the
// Request-Line as stored in RequestURI. For most requests, fields other than
// Path and RawQuery will be empty. (See RFC 7230, Section 5.3)
// Ref: https://github.com/golang/go/issues/28940
func (rc RequestCall) GetScheme() string {
	if rc.IsWebSocket() && rc.Request.TLS != nil {
		return SchemeWSS
	}

	if rc.IsWebSocket() {
		return SchemeWS
	}

	if rc.Request.TLS != nil {
		return SchemeHTTPS
	}

	return SchemeHTTP
}

// IsWebSocket - Checks whether a request is a websocket.
func (rc RequestCall) IsWebSocket() bool {
	return wsutil.IsWebSocketRequest(&rc.Request) // TODO: don't like the reference
}

// SendNotImplemented - Sends a 501 response status code.
func (rc RequestCall) SendNotImplemented(ctx context.Context) {
	rc.Response.SendNotImplemented()

	telemetry.From(ctx).RegisterStatusCode(http.StatusNotImplemented)
}

// SendMethodNotAllowed - Sends a 405 response status code.
func (rc RequestCall) SendMethodNotAllowed(ctx context.Context) {
	rc.Response.ForceWriteHeader(http.StatusMethodNotAllowed)

	telemetry.From(ctx).RegisterStatusCode(http.StatusMethodNotAllowed)
}

// SendNotModifiedResponse - Sends a 304 response status code.
func (rc RequestCall) SendNotModifiedResponse(ctx context.Context) {
	rc.Response.SendNotModifiedResponse()

	telemetry.From(ctx).RegisterStatusCode(http.StatusNotModified)
}

// SendResponse - Sends the response to the client.
func (rc RequestCall) SendResponse(ctx context.Context) {
	rc.Response.SendResponse()

	telemetry.From(ctx).RegisterStatusCode(rc.Response.StatusCode)
}
