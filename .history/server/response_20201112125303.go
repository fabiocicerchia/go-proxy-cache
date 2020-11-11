package server

import (
	"net/http"
)

type loggedResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Content    []byte
}

func newLoggedResponseWriter(w http.ResponseWriter) *loggedResponseWriter {
	var statusCode int
	var content []byte
	return &loggedResponseWriter{w, statusCode, content}
}

func (lrw *loggedResponseWriter) WriteHeader(statusCode int) {
	lrw.StatusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func (lrw *loggedResponseWriter) Write(p []byte) (int, error) {
	lrw.Content = p
	return lrw.ResponseWriter.Write(p)
}
