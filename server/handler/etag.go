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
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"

	"github.com/go-http-utils/fresh"
	"github.com/go-http-utils/headers"
	"github.com/yhat/wsutil"
)

// Version is this package's version.
const Version = "0.2.1"

type BufferedResponseWriter struct {
	http.ResponseWriter
	Hash             hash.Hash
	BufferedResponse *bytes.Buffer
	HashLen          int
	StatusCode       int
}

// func (bwr BufferedResponseWriter) Header() http.Header {
// 	// it'll reference the original writer, so no need to CopyHeaders later.
// 	return bwr.ResponseWriter.Header()
// }

func (bwr *BufferedResponseWriter) WriteHeader(status int) {
	bwr.StatusCode = status
}

func (bwr *BufferedResponseWriter) Write(b []byte) (int, error) {
	// bytes.Buffer.Write(b) always return (len(b), nil), so just
	// ignore the return values.
	bwr.BufferedResponse.Write(b)

	l, err := bwr.Hash.Write(b)
	bwr.HashLen += l

	return l, err
}

// GetETag - Returns the ETag value.
func (bwr BufferedResponseWriter) GetETag(weak bool) string {
	etagWeakPrefix := ""
	if weak {
		etagWeakPrefix = "W/"
	}

	return fmt.Sprintf("%s%d-%s", etagWeakPrefix, bwr.HashLen, hex.EncodeToString(bwr.Hash.Sum(nil)))
}

func (bwr *BufferedResponseWriter) SetETag(weak bool) {
	bwr.ResponseWriter.Header().Set(headers.ETag, bwr.GetETag(weak))
}

func (bwr BufferedResponseWriter) MustServeOriginalResponse() bool {
	// return bwr.Hash == nil || // no hash has been computed (maybe no Write has been invoked)
	return bwr.ResponseWriter.Header().Get(headers.ETag) != "" || // there's already an ETag from upstream
		(bwr.StatusCode < 200 || bwr.StatusCode > 299) || // response is not successful (2xx)
		bwr.StatusCode == http.StatusNoContent || // response is without countent (204)
		bwr.BufferedResponse.Len() == 0 // there is no buffered content (maybe no Write has been invoked)
}

func (bwr BufferedResponseWriter) SendResponse() {
	if bwr.StatusCode == 0 {
		bwr.StatusCode = http.StatusOK // TODO: WHY?
	}

	bwr.ResponseWriter.WriteHeader(bwr.StatusCode)
	bwr.ResponseWriter.Write(bwr.BufferedResponse.Bytes())
}

func ConditionalETag(h http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		HandleETagRequest(res, req, h)
	})
}

// HandleETagRequest - Add HTTP header ETag only on HTTP(S) requests.
func HandleETagRequest(res http.ResponseWriter, req *http.Request, h http.Handler) {
	// ETag wrapper doesn't work well with WebSocket and HTTP/2.
	if wsutil.IsWebSocketRequest(req) || req.ProtoMajor == 2 {
		h.ServeHTTP(res, req)
		return
	}

	weak := false

	bwr := BufferedResponseWriter{
		ResponseWriter:   res,
		Hash:             sha1.New(),
		BufferedResponse: bytes.NewBuffer(nil),
	}
	// Start buffering the response.
	h.ServeHTTP(&bwr, req)

	// Serve existing response.
	if bwr.MustServeOriginalResponse() {
		bwr.SendResponse()
		return
	}

	bwr.SetETag(weak)

	// Send 304 Not Modified.
	if fresh.IsFresh(req.Header, res.Header()) {
		res.WriteHeader(http.StatusNotModified)
		res.Write(nil)
		return
	}

	// Serve response with ETag header.
	bwr.SendResponse()
}
