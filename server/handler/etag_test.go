//go:build all || functional
// +build all functional

package handler_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"regexp"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

func TestGetResponseWithETagWithHTTP2(t *testing.T) {
	initLogs()

	reqMock := http.Request{
		Proto:      "HTTPS",
		ProtoMajor: 2,
		Method:     "GET",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	}

	proxyUrl := &url.URL{Scheme: "https", Host: "www.w3.org"}
	proxy := httputil.NewSingleHostReverseProxy(proxyUrl)

	reqID := "TestGetResponseWithETag"
	rr := httptest.NewRecorder()
	rcMock := handler.RequestCall{
		ReqID:    reqID,
		Response: response.NewLoggedResponseWriter(rr, reqID),
		Request:  reqMock,
	}

	tracingSpan := opentracing.GlobalTracer().StartSpan("")
	defer tracingSpan.Finish()
	ctx := opentracing.ContextWithSpan(reqMock.Context(), tracingSpan)

	serveNotModified := rcMock.GetResponseWithETag(ctx, proxy)

	assert.False(t, serveNotModified)
}

func TestGetResponseWithETagWithExistingETag(t *testing.T) {
	initLogs()

	// Page with actual ETag
	reqMock := http.Request{
		Proto:      "HTTPS",
		ProtoMajor: 2,
		Method:     "GET",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Scheme: "https", Host: "www.w3.org", Path: "/"},
		Header: http.Header{
			"Host": []string{"www.w3.org"},
		},
	}

	proxyUrl := &url.URL{Scheme: "https", Host: "www.w3.org"}
	proxy := httputil.NewSingleHostReverseProxy(proxyUrl)

	reqID := "TestGetResponseWithETag"
	rr := httptest.NewRecorder()
	rr.Header().Add("ETag", "TestGetResponseWithETagWithExistingETag")
	rcMock := handler.RequestCall{
		ReqID:    reqID,
		Response: response.NewLoggedResponseWriter(rr, reqID),
		Request:  reqMock,
	}

	tracingSpan := opentracing.GlobalTracer().StartSpan("")
	defer tracingSpan.Finish()
	ctx := opentracing.ContextWithSpan(reqMock.Context(), tracingSpan)

	serveNotModified := rcMock.GetResponseWithETag(ctx, proxy)

	assert.False(t, serveNotModified)
	assert.Equal(t, "TestGetResponseWithETagWithExistingETag", rr.Header().Get("ETag"))
}

func TestGetResponseWithETagGeneratedInternally(t *testing.T) {
	initLogs()

	// Page without actual ETag
	reqMock := http.Request{
		Proto:      "HTTPS",
		Method:     "GET",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Scheme: "https", Host: "www.google.com", Path: "/"},
		Header: http.Header{
			"Host": []string{"www.google.com"},
		},
	}
	reqMock.TLS = &tls.ConnectionState{} // mock a fake https

	proxyUrl := &url.URL{Scheme: "https", Host: "www.google.com"}
	proxy := httputil.NewSingleHostReverseProxy(proxyUrl)

	reqID := "TestGetResponseWithETagGeneratedInternally"
	rr := httptest.NewRecorder()
	rcMock := handler.RequestCall{
		ReqID:    reqID,
		Response: response.NewLoggedResponseWriter(rr, reqID),
		Request:  reqMock,
	}

	tracingSpan := opentracing.GlobalTracer().StartSpan("")
	defer tracingSpan.Finish()
	ctx := opentracing.ContextWithSpan(reqMock.Context(), tracingSpan)

	serveNotModified := rcMock.GetResponseWithETag(ctx, proxy)

	assert.False(t, serveNotModified)
	assert.Regexp(t, regexp.MustCompile(`^\"[0-9]+-[0-9a-f]{40}\"$`), rr.Header().Get("ETag"))
}

func TestGetResponseWithETagGeneratedInternallyAndFresh(t *testing.T) {
	initLogs()

	// Page without actual ETag
	reqMock := http.Request{
		Proto:      "HTTPS",
		Method:     "GET",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Scheme: "https", Host: "www.google.com", Path: "/"},
		Header: http.Header{
			"Host":          []string{"www.google.com"},
			"If-None-Match": []string{"*"},
		},
	}
	reqMock.TLS = &tls.ConnectionState{} // mock a fake https

	proxyUrl := &url.URL{Scheme: "https", Host: "www.google.com"}
	proxy := httputil.NewSingleHostReverseProxy(proxyUrl)

	reqID := "TestGetResponseWithETagGeneratedInternally"
	rr := httptest.NewRecorder()
	rcMock := handler.RequestCall{
		ReqID:    reqID,
		Response: response.NewLoggedResponseWriter(rr, reqID),
		Request:  reqMock,
	}

	tracingSpan := opentracing.GlobalTracer().StartSpan("")
	defer tracingSpan.Finish()
	ctx := opentracing.ContextWithSpan(reqMock.Context(), tracingSpan)

	serveNotModified := rcMock.GetResponseWithETag(ctx, proxy)

	assert.True(t, serveNotModified)
	assert.Regexp(t, regexp.MustCompile(`^\"[0-9]+-[0-9a-f]{40}\"$`), rr.Header().Get("ETag"))
}
