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
	"net/http/httputil"

	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	log "github.com/sirupsen/logrus"
	"github.com/yhat/wsutil"

	"github.com/go-http-utils/fresh"
)

// HandleRequestWithETag - Add HTTP header ETag only on HTTP(S) requests.
func GetResponseWithETag(res *response.LoggedResponseWriter, req *http.Request, proxy *httputil.ReverseProxy) (serveNotModified bool) {
	// ETag wrapper doesn't work well with WebSocket and HTTP/2.
	res.IsBuffered = !wsutil.IsWebSocketRequest(req) && req.ProtoMajor != 2

	// Start buffering the response.
	proxy.ServeHTTP(res, req)

	// Serve existing response.
	if res.MustServeOriginalResponse(req) {
		log.Info("Serving original response as cannot be handled with ETag.")
		return false
	}

	res.SetETag(false)

	// Send 304 Not Modified.
	if fresh.IsFresh(req.Header, res.Header()) {
		return true
	}

	// Serve response with ETag header.
	return false
}
