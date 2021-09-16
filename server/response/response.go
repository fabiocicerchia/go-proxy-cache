package response

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"net"
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/server/tracing"
	"github.com/go-http-utils/headers"
)

var errHijackNotSupported = errors.New("hijack not supported")

// LoggedResponseWriter - Decorator for http.ResponseWriter.
type LoggedResponseWriter struct {
	http.ResponseWriter
	http.Hijacker

	ReqID          string
	statusCodeSent bool
	StatusCode     int
	Content        DataChunks

	// GZip
	GZipResponse *gzip.Writer

	// ETag
	hash    hash.Hash
	hashLen int
}

// NewLoggedResponseWriter - Creates new instance of ResponseWriter.
func NewLoggedResponseWriter(w http.ResponseWriter, reqID string) *LoggedResponseWriter {
	lwr := &LoggedResponseWriter{
		ReqID:          reqID,
		ResponseWriter: w,
		hash:           sha1.New(),
	}
	lwr.Reset()

	return lwr
}

// Hijack lets the caller take over the connection.
func (lwr *LoggedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := lwr.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errHijackNotSupported
	}

	return hj.Hijack()
}

// Reset - Reset the stored content of LoggedResponseWriter.
func (lwr *LoggedResponseWriter) Reset() {
	lwr.StatusCode = 0
	lwr.Content = make(DataChunks, 0)
}

// WriteHeader - ResponseWriter's WriteHeader method decorator.
func (lwr *LoggedResponseWriter) WriteHeader(statusCode int) {
	lwr.statusCodeSent = true
	lwr.StatusCode = statusCode

	// no sending to ResponseWriter as it is buffered either for ETag or GZip support.
}

// ForceWriteHeader - Send statusCode right away.
func (lwr *LoggedResponseWriter) ForceWriteHeader(statusCode int) {
	lwr.WriteHeader(statusCode)

	lwr.ResponseWriter.WriteHeader(statusCode)
}

// SendNotImplemented - Send 501 Not Implemented.
func (lwr *LoggedResponseWriter) SendNotImplemented() {
	lwr.ForceWriteHeader(http.StatusNotImplemented)
}

// Write - ResponseWriter's Write method decorator.
func (lwr *LoggedResponseWriter) Write(p []byte) (int, error) {
	if !lwr.statusCodeSent && lwr.StatusCode == 0 {
		tracing.AddEventsToSpan(tracing.SpanFromContext(context.Background()), "response.missing_status_code", map[string]string{})
		lwr.GetLogger().Warning("No status code has been set before sending data, fallback on 200 OK.")

		// This is exactly what Go would also do if it hasn't been written yet.
		lwr.StatusCode = http.StatusOK
	}

	lwr.Content = append(lwr.Content, []byte{})
	chunk := len(lwr.Content) - 1
	lwr.Content[chunk] = append(lwr.Content[chunk], p...)

	// gzip
	if lwr.GZipResponse != nil {
		if lwr.ResponseWriter.Header().Get(headers.ContentType) == "" {
			// If no content type, apply sniffing algorithm to un-gzipped body.
			lwr.ResponseWriter.Header().Set(headers.ContentType, http.DetectContentType(p))
		}

		lwr.GZipResponse.Write(p)
	}

	// etag
	l, err := lwr.hash.Write(p)
	lwr.hashLen += l

	// no sending to ResponseWriter as it is buffered either for ETag or GZip support.
	return l, err
}

// ForceWrite - Send content right away.
func (lwr *LoggedResponseWriter) ForceWrite(p []byte) (int, error) {
	lwr.Write(p)

	return lwr.ResponseWriter.Write(p)
}

// CopyHeaders - Adds the headers to the response.
func (lwr *LoggedResponseWriter) CopyHeaders(src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			lwr.ResponseWriter.Header().Add(k, v)
		}
	}
}

// WriteBody - Sends the body to the client (forced sent).
func (lwr *LoggedResponseWriter) WriteBody(page string) bool {
	pageByte := []byte(page)
	sent, err := lwr.ResponseWriter.Write(pageByte)

	return sent > 0 && err == nil
}

// SendResponse - Write the Response.
func (lwr LoggedResponseWriter) SendResponse() {
	// TODO! Get extra behaviour from ServeCachedResponse
	lwr.ResponseWriter.WriteHeader(lwr.StatusCode)

	// Generate GZip.
	// lwr.GZipResponse.Close() will write some data even if no data has been written.
	// StatusNotModified and StatusNoContent shouldn't have a body, so no triggering Close().
	if lwr.GZipResponse != nil && lwr.StatusCode != http.StatusNotModified && lwr.StatusCode != http.StatusNoContent {
		// In this way it'll write in a nested LoggedResponseWriter so it can
		// catch the binary data.
		lwr.GZipResponse.Close()
	}

	// Serve content.
	lwr.ResponseWriter.Write(lwr.Content.Bytes())
}

// ETAG ------------------------------------------------------------------------

// GetETag - Returns the ETag value.
func (lwr LoggedResponseWriter) GetETag(weak bool) string {
	etagWeakPrefix := ""
	if weak {
		etagWeakPrefix = "W/"
	}

	return fmt.Sprintf(`"%s%d-%s"`, etagWeakPrefix, lwr.hashLen, hex.EncodeToString(lwr.hash.Sum(nil)))
}

// SetETag - Set the ETag HTTP Header.
func (lwr *LoggedResponseWriter) SetETag(weak bool) {
	lwr.ResponseWriter.Header().Set(headers.ETag, lwr.GetETag(weak))
}

// MustServeOriginalResponse - Check whether an ETag could be added.
func (lwr LoggedResponseWriter) MustServeOriginalResponse(ctx context.Context, req *http.Request) bool {
	tracing.SpanFromContext(ctx).
		SetTag("response.must_serve_original_response.no_hash_computed", lwr.hash == nil).
		SetTag("response.must_serve_original_response.etag_present", lwr.ResponseWriter.Header().Get(headers.ETag)).
		SetTag("response.must_serve_original_response.etag_already_present", lwr.ResponseWriter.Header().Get(headers.ETag) != "").
		SetTag("response.must_serve_original_response.response_status_code", lwr.StatusCode).
		SetTag("response.must_serve_original_response.response_not_2xx", (lwr.StatusCode < http.StatusOK || lwr.StatusCode >= http.StatusMultipleChoices)).
		SetTag("response.must_serve_original_response.response_204", lwr.StatusCode == http.StatusNoContent).
		SetTag("response.must_serve_original_response.no_buffered_content", len(lwr.Content) == 0)

	lwr.GetLogger().Debugf("MustServerOriginalResponse - no hash has been computed (maybe no Write has been invoked): %v", lwr.hash == nil)
	lwr.GetLogger().Debugf("MustServerOriginalResponse - there's already an ETag from upstream: %v (%s)", lwr.ResponseWriter.Header().Get(headers.ETag) != "", lwr.ResponseWriter.Header().Get(headers.ETag))
	lwr.GetLogger().Debugf("MustServerOriginalResponse - response is not successful (2xx): %v (%d)", (lwr.StatusCode < http.StatusOK || lwr.StatusCode >= http.StatusMultipleChoices), lwr.StatusCode)
	lwr.GetLogger().Debugf("MustServerOriginalResponse - response is without content (204): %v", lwr.StatusCode == http.StatusNoContent)
	lwr.GetLogger().Debugf("MustServerOriginalResponse - there is no buffered content (maybe no Write has been invoked): %v", len(lwr.Content) == 0)

	return lwr.hash == nil || // no hash has been computed (maybe no Write has been invoked)
		lwr.ResponseWriter.Header().Get(headers.ETag) != "" || // there's already an ETag from upstream
		(lwr.StatusCode < http.StatusOK || lwr.StatusCode >= http.StatusMultipleChoices) || // response is not successful (2xx)
		lwr.StatusCode == http.StatusNoContent || // response is without content (204)
		len(lwr.Content) == 0 // there is no buffered content (maybe no Write has been invoked)
}

// SendNotModifiedResponse - Write the 304 Response.
func (lwr LoggedResponseWriter) SendNotModifiedResponse() {
	lwr.ResponseWriter.WriteHeader(http.StatusNotModified)
	lwr.ResponseWriter.Write(nil)
}

// GZIP ------------------------------------------------------------------------
func (lwr *LoggedResponseWriter) InitGZipBuffer() {
	lwrGzip := &LoggedResponseWriter{ResponseWriter: lwr.ResponseWriter}
	lwr.GZipResponse = gzip.NewWriter(lwrGzip)
}
