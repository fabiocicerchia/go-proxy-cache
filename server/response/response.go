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
	"net/http"
)

// CacheStatusHeader - HTTP Header for showing cache status
const CacheStatusHeader = "X-Go-Proxy-Cache-Status"

// CacheStatusHeaderHit - Cache status HIT for HTTP Header X-Go-Proxy-Cache-Status
const CacheStatusHeaderHit = "HIT"

// CacheStatusHeaderMiss - Cache status MISS for HTTP Header X-Go-Proxy-Cache-Status
const CacheStatusHeaderMiss = "MISS"

// LoggedResponseWriter - Decorator for http.ResponseWriter
type LoggedResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Content    [][]byte
}

// NewLoggedResponseWriter - Creates new instance of ResponseWriter.
func NewLoggedResponseWriter(w http.ResponseWriter) *LoggedResponseWriter {
	lwr := &LoggedResponseWriter{ResponseWriter: w}
	lwr.Reset()
	return lwr
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
func CopyHeaders(dst http.Header, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// WriteBody - Sends the body to the client.
func WriteBody(rw http.ResponseWriter, page string) bool {
	pageByte := []byte(page)
	sent, err := rw.Write(pageByte)

	return sent > 0 && err == nil
}
