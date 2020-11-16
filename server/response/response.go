package response

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

func (lwr *LoggedResponseWriter) WriteHeader(statusCode int) {
	lwr.StatusCode = statusCode
	lwr.ResponseWriter.WriteHeader(statusCode)
}

func (lwr *LoggedResponseWriter) Write(p []byte) (int, error) {
	lwr.Content = append(lwr.Content, p...)
	return lwr.ResponseWriter.Write(p)
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
