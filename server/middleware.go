package server

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

	"github.com/go-http-utils/etag"
	"github.com/yhat/wsutil"
)

func ConditionalETag(h http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		// ETag wrapper doesn't work well with WebSocket and HTTP/2.
		if !wsutil.IsWebSocketRequest(req) && req.ProtoMajor != 2 {
			etagHandler := etag.Handler(h, false)
			etagHandler.ServeHTTP(res, req)
		} else {
			h.ServeHTTP(res, req)
		}
	})
}
