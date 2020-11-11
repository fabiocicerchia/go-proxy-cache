package server

import (
	"net/http"
)

type loggedResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Content    []byte
}

func LoggedResponseWriter(w http.ResponseWriter) *loggedResponseWriter {
	return &loggedResponseWriter{w, http.StatusOK}
}

func (lrw *loggedResponseWriter) WriteHeader(statusCode int) {
	lrw.StatusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func (lrw *loggedResponseWriter) Write(p []byte) (int, error) {
	lrw.Content = p
	return lrw.ResponseWriter.Write(p)
}
