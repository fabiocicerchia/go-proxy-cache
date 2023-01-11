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
	"net/http"
	"strings"

	"github.com/go-http-utils/headers"

	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

// WrapResponseForGZip - Add HTTP header ETag only on HTTP(S) requests.
func WrapResponseForGZip(res *response.LoggedResponseWriter, req *http.Request) {
	if !strings.Contains(req.Header.Get(headers.AcceptEncoding), "gzip") {
		return
	}

	res.Header().Set(headers.ContentEncoding, "gzip")
}
