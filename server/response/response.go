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
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"net"
	"net/http"

	"github.com/go-http-utils/headers"
	log "github.com/sirupsen/logrus"
)

var errHijackNotSupported = errors.New("hijack not supported")

// CacheStatusHeader - HTTP Header for showing cache status.
const CacheStatusHeader = "X-Go-Proxy-Cache-Status"

// CacheStatusHeader - HTTP Header for showing cache status.
const CacheBypassHeader = "X-Go-Proxy-Cache-Force-Fresh"

// CacheStatusHeaderHit - Cache status HIT for HTTP Header X-Go-Proxy-Cache-Status.
const CacheStatusHeaderHit = "HIT"

// CacheStatusHeaderMiss - Cache status MISS for HTTP Header X-Go-Proxy-Cache-Status.
const CacheStatusHeaderMiss = "MISS"

// CacheStatusHeaderStale - Cache status STALE for HTTP Header X-Go-Proxy-Cache-Status.
const CacheStatusHeaderStale = "STALE"

// type parsedContentType struct {
// 	mediaType string
// 	params    map[string]string
// }

// LoggedResponseWriter - Decorator for http.ResponseWriter.
type LoggedResponseWriter struct {
	http.ResponseWriter
	http.Hijacker
	statusCodeSent bool
	StatusCode     int
	Content        [][]byte

	// GZip + ETag
	IsBuffered   bool
	GZipResponse *gzip.Writer

	// ETag
	BufferedResponse *bytes.Buffer
	hash             hash.Hash
	hashLen          int
}

// NewLoggedResponseWriter - Creates new instance of ResponseWriter.
func NewLoggedResponseWriter(w http.ResponseWriter) *LoggedResponseWriter {
	lwr := &LoggedResponseWriter{
		IsBuffered:       true, // TODO: REmove from other places
		ResponseWriter:   w,
		hash:             sha1.New(),
		BufferedResponse: bytes.NewBuffer(nil),
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
	lwr.Content = make([][]byte, 0)
}

// WriteHeader - ResponseWriter's WriteHeader method decorator.
func (lwr *LoggedResponseWriter) WriteHeader(statusCode int) {
	lwr.statusCodeSent = true
	lwr.StatusCode = statusCode

	if !lwr.IsBuffered {
		lwr.ResponseWriter.WriteHeader(statusCode)
	}
}

// SendNotImplemented - Send 501 Not Implemented.
func (lwr *LoggedResponseWriter) SendNotImplemented() {
	lwr.IsBuffered = false
	lwr.WriteHeader(http.StatusNotImplemented)
}

// Write - ResponseWriter's Write method decorator.
func (lwr *LoggedResponseWriter) Write(p []byte) (int, error) {
	if !lwr.statusCodeSent && lwr.StatusCode == 0 {
		log.Warning("No status code has been set before sending data, fallback on 200 OK.")
		// This is exactly what Go would also do if it hasn't been written yet.
		lwr.StatusCode = http.StatusOK
	}

	lwr.Content = append(lwr.Content, []byte{})
	chunk := len(lwr.Content) - 1
	lwr.Content[chunk] = append(lwr.Content[chunk], p...)

	// gzip
	if lwr.GZipResponse != nil {
		if lwr.Header().Get(headers.ContentType) == "" {
			// If no content type, apply sniffing algorithm to un-gzipped body.
			lwr.Header().Set(headers.ContentType, http.DetectContentType(p))
		}

		lwr.GZipResponse.Write(p)
	}

	// etag
	if lwr.IsBuffered {
		// bytes.Buffer.Write(b) always return (len(b), nil), so just
		// ignore the return values.
		lwr.BufferedResponse.Write(p) // TODO: use lwr.Content instead of BufferedResponse

		l, err := lwr.hash.Write(p)
		lwr.hashLen += l

		return l, err
	}

	return lwr.ResponseWriter.Write(p)
}

// CopyHeaders - Adds the headers to the response.
func (lwr *LoggedResponseWriter) CopyHeaders(src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			lwr.Header().Add(k, v)
		}
	}
}

// WriteBody - Sends the body to the client.
func (lwr *LoggedResponseWriter) WriteBody(page string) bool {
	pageByte := []byte(page)
	sent, err := lwr.ResponseWriter.Write(pageByte)

	return sent > 0 && err == nil
}

// ETAG ------------------------------------------------------------------------

// GetETag - Returns the ETag value.
func (lwr LoggedResponseWriter) GetETag(weak bool) string {
	etagWeakPrefix := ""
	if weak {
		etagWeakPrefix = "W/"
	}

	return fmt.Sprintf("%s%d-%s", etagWeakPrefix, lwr.hashLen, hex.EncodeToString(lwr.hash.Sum(nil)))
}

// SetETag - Set the ETag HTTP Header.
func (lwr *LoggedResponseWriter) SetETag(weak bool) {
	lwr.ResponseWriter.Header().Set(headers.ETag, lwr.GetETag(weak))
}

// MustServeOriginalResponse - Check whether an ETag could be added.
func (lwr LoggedResponseWriter) MustServeOriginalResponse(req *http.Request) bool {
	return lwr.hash == nil || // no hash has been computed (maybe no Write has been invoked)
		lwr.ResponseWriter.Header().Get(headers.ETag) != "" || // there's already an ETag from upstream
		(lwr.StatusCode < http.StatusOK || lwr.StatusCode >= http.StatusMultipleChoices) || // response is not successful (2xx)
		lwr.StatusCode == http.StatusNoContent || // response is without content (204)
		lwr.BufferedResponse.Len() == 0 // there is no buffered content (maybe no Write has been invoked)
}

// SendNotModifiedResponse - Write the 304 Response.
func (lwr LoggedResponseWriter) SendNotModifiedResponse() {
	lwr.ResponseWriter.WriteHeader(http.StatusNotModified)
	lwr.ResponseWriter.Write(nil)
}

// SendResponse - Write the Response.
func (lwr LoggedResponseWriter) SendResponse() {
	// TODO: Get extra behaviour from ServeCachedResponse
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
	lwr.ResponseWriter.Write(lwr.BufferedResponse.Bytes())
}

// GZIP ------------------------------------------------------------------------
func (lwr LoggedResponseWriter) InitGZipBuffer() {
	lwrGzip := &LoggedResponseWriter{
		IsBuffered:     true,
		ResponseWriter: lwr.ResponseWriter,
	}

	lwr.GZipResponse = gzip.NewWriter(lwrGzip)
}
