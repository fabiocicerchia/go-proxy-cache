package handler_test

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
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

// HandleRequestWithETag - Add HTTP header ETag only on HTTP(S) requests.
func TestWrapResponseForGZipWhenNoAcceptEncoding(t *testing.T) {
	initLogs()

	reqMock := http.Request{
		Proto:      "HTTPS",
		Method:     "GET",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host":            []string{"localhost"},
			"Accept-Encoding": []string{""},
		},
	}

	reqID := "TestWrapResponseForGZipWhenNoAcceptEncoding"
	rr := httptest.NewRecorder()
	res := response.NewLoggedResponseWriter(rr, reqID)

	handler.WrapResponseForGZip(res, &reqMock)

	assert.Equal(t, "", rr.Header().Get("Content-Encoding"))
}

// HandleRequestWithETag - Add HTTP header ETag only on HTTP(S) requests.
func TestWrapResponseForGZipWhenAcceptEncodingGZip(t *testing.T) {
	initLogs()

	reqMock := http.Request{
		Proto:      "HTTPS",
		Method:     "GET",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host":            []string{"localhost"},
			"Accept-Encoding": []string{"gzip"},
		},
	}

	reqID := "TestWrapResponseForGZipWhenNoAcceptEncoding"
	rr := httptest.NewRecorder()
	res := response.NewLoggedResponseWriter(rr, reqID)

	handler.WrapResponseForGZip(res, &reqMock)

	assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))
}
