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
	"net/url"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/yhat/wsutil"
	"go.opentelemetry.io/otel/trace"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/tracing"
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
	TracingSpan  trace.Span // TODO: REMOVE
}

// GetLogger - Get logger instance with RequestID.
func (rc RequestCall) GetLogger() *log.Entry {
	return log.WithFields(log.Fields{
		"ReqID": rc.ReqID,
	})
}

// IsLegitRequest - Check whether a request is bound on the right Host and Port.
func (rc RequestCall) IsLegitRequest(listeningPort string) bool {
	hostMatch := rc.DomainConfig.Server.Upstream.Host == rc.GetHostname()
	legitPort := isLegitPort(rc.DomainConfig.Server.Port, listeningPort)

	tracing.AddTagsToSpan(tracing.SpanFromContext(rc.Request.Context()), map[string]string{
		"request.is_legit.hostname_matches": fmt.Sprintf("%v", hostMatch),
		"request.is_legit.port_matches":     fmt.Sprintf("%v", legitPort),
		"request.is_legit.req_hostname":     rc.GetHostname(),
		"request.is_legit.req_port":         listeningPort,
		"request.is_legit.conf_hostname":    rc.DomainConfig.Server.Upstream.Host,
		"request.is_legit.conf_port":        fmt.Sprintf("%v", rc.DomainConfig.Server.Port),
	})

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
// Path and RawQuery will be empty. (TracingSpan RFC 7230, Section 5.3)
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

func (rc RequestCall) SendNotImplemented() {
	rc.Response.SendNotImplemented()

	tracing.AddTagsToSpan(tracing.SpanFromContext(rc.Request.Context()), map[string]string{
		"response.status_code": strconv.Itoa(http.StatusNotImplemented),
	})
}

func (rc RequestCall) SendMethodNotAllowed() {
	rc.Response.ForceWriteHeader(http.StatusMethodNotAllowed)

	tracing.AddTagsToSpan(tracing.SpanFromContext(rc.Request.Context()), map[string]string{
		"response.status_code": strconv.Itoa(http.StatusMethodNotAllowed),
	})
}

func (rc RequestCall) SendNotModifiedResponse() {
	rc.Response.SendNotModifiedResponse()

	tracing.AddTagsToSpan(tracing.SpanFromContext(rc.Request.Context()), map[string]string{
		"response.status_code": strconv.Itoa(http.StatusNotModified),
	})
}

func (rc RequestCall) SendResponse() {
	rc.Response.SendResponse()

	tracing.AddTagsToSpan(tracing.SpanFromContext(rc.Request.Context()), map[string]string{
		"response.status_code": strconv.Itoa(rc.Response.StatusCode),
	})
}
