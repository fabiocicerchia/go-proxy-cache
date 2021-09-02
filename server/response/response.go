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
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"net"
	"net/http"

	"github.com/go-http-utils/headers"
)

var errHijackNotSupported = errors.New("hijack not supported")

// CacheStatusHeader - HTTP Header for showing cache status.
const CacheStatusHeader = "X-Go-Proxy-Cache-Status"

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
	StatusCode int
	Content    [][]byte

	// ETag
	IsBuffered       bool
	hash             hash.Hash
	bufferedResponse *bytes.Buffer
	hashLen          int

	// GZip
	// index        int // Index for gzipWriterPools.
	// gw           *gzip.Writer
	// code         int                 // Saves the WriteHeader value.
	// minSize      int                 // Specifed the minimum response size to gzip. If the response length is bigger than this value, it is compressed.
	// buf          []byte              // Holds the first part of the write before reaching the minSize or the end of the write.
	// ignore       bool                // If true, then we immediately passthru writes to the underlying ResponseWriter.
	// contentTypes []parsedContentType // Only compress if the response is one of these content-types. All are accepted if empty.
}

// NewLoggedResponseWriter - Creates new instance of ResponseWriter.
func NewLoggedResponseWriter(w http.ResponseWriter) *LoggedResponseWriter {
	lwr := &LoggedResponseWriter{
		ResponseWriter:   w,
		hash:             sha1.New(),
		bufferedResponse: bytes.NewBuffer(nil),
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
	lwr.StatusCode = statusCode

	if !lwr.IsBuffered {
		lwr.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write - ResponseWriter's Write method decorator.
func (lwr *LoggedResponseWriter) Write(p []byte) (int, error) {
	lwr.Content = append(lwr.Content, []byte{})
	chunk := len(lwr.Content) - 1
	lwr.Content[chunk] = append(lwr.Content[chunk], p...)

	if lwr.IsBuffered {
		// bytes.Buffer.Write(b) always return (len(b), nil), so just
		// ignore the return values.
		lwr.bufferedResponse.Write(p)

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
		(lwr.StatusCode < 200 || lwr.StatusCode > 299) || // response is not successful (2xx)
		lwr.StatusCode == http.StatusNoContent || // response is without content (204)
		lwr.bufferedResponse.Len() == 0 // there is no buffered content (maybe no Write has been invoked)
}

// SendNotMofifiedResponse - Write the 304 Response.
func (lwr LoggedResponseWriter) SendNotMofifiedResponse() {
	lwr.ResponseWriter.WriteHeader(http.StatusNotModified)
	lwr.ResponseWriter.Write(nil)
}

// SendResponse - Write the Response.
func (lwr LoggedResponseWriter) SendResponse() {
	if lwr.StatusCode == 0 {
		lwr.StatusCode = http.StatusOK // TODO: WHY?
	}

	// TODO: Get extra behaviour from ServeCachedResponse
	lwr.ResponseWriter.WriteHeader(lwr.StatusCode)
	lwr.ResponseWriter.Write(lwr.bufferedResponse.Bytes())
}
