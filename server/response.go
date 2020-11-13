package server

import (
	"net/http"
)

type LoggedResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Content    []byte
}

func NewLoggedResponseWriter(w http.ResponseWriter) *LoggedResponseWriter {
	return &LoggedResponseWriter{w, 0, []byte("")}
}

func (lrw *LoggedResponseWriter) WriteHeader(statusCode int) {
	lrw.StatusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func (lrw *LoggedResponseWriter) Write(p []byte) (int, error) {
	lrw.Content = append(lrw.Content, p...)
	return lrw.ResponseWriter.Write(p)
}
