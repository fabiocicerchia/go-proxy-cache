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
	"strings"

	"github.com/yhat/wsutil"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

var SchemeHTTP string = "http"
var SchemeHTTPS string = "https"
var SchemeWS string = "ws"
var SchemeWSS string = "wss"

// RequestCall - Main object containing request and response.
type RequestCall struct {
	Response     *response.LoggedResponseWriter
	Request      *http.Request
	DomainConfig *config.Configuration
}

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
	return wsutil.IsWebSocketRequest(rc.Request)
}
