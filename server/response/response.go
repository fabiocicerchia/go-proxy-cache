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
	"errors"
	"net"
	"net/http"
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

// LoggedResponseWriter - Decorator for http.ResponseWriter.
type LoggedResponseWriter struct {
	http.ResponseWriter
	http.Hijacker
	StatusCode int
	Content    [][]byte
}

// NewLoggedResponseWriter - Creates new instance of ResponseWriter.
func NewLoggedResponseWriter(w http.ResponseWriter) *LoggedResponseWriter {
	lwr := &LoggedResponseWriter{ResponseWriter: w}
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
	lwr.ResponseWriter.WriteHeader(statusCode)
}

// Write - ResponseWriter's Write method decorator.
func (lwr *LoggedResponseWriter) Write(p []byte) (int, error) {
	lwr.Content = append(lwr.Content, []byte{})
	chunk := len(lwr.Content) - 1
	lwr.Content[chunk] = append(lwr.Content[chunk], p...)

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
