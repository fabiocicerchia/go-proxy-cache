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

// TODO: coverage
func CopyHeaders(rw http.ResponseWriter, headers map[string]interface{}) {
	for k, v := range headers {
		rw.Header().Set(k, string(v.([]byte)))
	}
}

// TODO: coverage
func Flush(rw http.ResponseWriter) {
	if fl, ok := rw.(http.Flusher); ok {
		fl.Flush()
	}
}

func WriteBody(rw http.ResponseWriter, page string) bool {
	pageByte := []byte(page)
	sent, err := rw.Write(pageByte)
	// try again
	if sent == 0 && err != nil {
		// TODO: log
		sent, err = rw.Write(pageByte)
		return sent > 0 && err == nil
	}

	return true
}
