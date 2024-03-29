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
	"net/http/httputil"

	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/go-http-utils/fresh"
	"github.com/yhat/wsutil"
)

// HttpVersion2 - The value for the HTTP/2 protocol.
const HttpVersion2 = 2

// GetResponseWithETag - Add HTTP header ETag only on HTTP(S) requests.
func (rc RequestCall) GetResponseWithETag(ctx context.Context, proxy *httputil.ReverseProxy) (serveNotModified bool) {
	// Start buffering the response.
	proxy.ServeHTTP(rc.Response, &rc.Request)

	// ETag wrapper doesn't work well with WebSocket and HTTP/2.
	if wsutil.IsWebSocketRequest(&rc.Request) || rc.Request.ProtoMajor == HttpVersion2 {
		telemetry.From(ctx).RegisterEvent("request.etag.not_supported")
		rc.GetLogger().Info("Current request doesn't support ETag.")

		// Serve existing response.
		return false
	}

	// Serve existing response.
	if rc.Response.MustServeOriginalResponse(ctx, &rc.Request) {
		telemetry.From(ctx).RegisterEvent("request.etag.serve_original")
		rc.GetLogger().Info("Serving original response as cannot be handled with ETag.")

		return false
	}

	rc.Response.SetETag(false)

	// Send 304 Not Modified when response is still fresh.
	// Serve response with ETag header, in case response is not fresh anymore.
	return fresh.IsFresh(rc.Request.Header, rc.Response.Header())
}
