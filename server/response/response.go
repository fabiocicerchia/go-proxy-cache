package response

import (
	"log"
	"net/http"
)

// LoggedResponseWriter - Decorator for http.ResponseWriter
type LoggedResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Content    []byte
}

// NewLoggedResponseWriter - Creates new instance of ResponseWriter.
func NewLoggedResponseWriter(w http.ResponseWriter) *LoggedResponseWriter {
	return &LoggedResponseWriter{w, 0, []byte("")}
}

// WriteHeader - ResponseWriter's WriteHeader method decorator.
func (lwr *LoggedResponseWriter) WriteHeader(statusCode int) {
	lwr.StatusCode = statusCode
	lwr.ResponseWriter.WriteHeader(statusCode)
}

// Write - ResponseWriter's Write method decorator.
func (lwr *LoggedResponseWriter) Write(p []byte) (int, error) {
	lwr.Content = append(lwr.Content, p...)
	return lwr.ResponseWriter.Write(p)
}

// CopyHeaders - Adds the headers to the response.
func CopyHeaders(rw http.ResponseWriter, headers map[string]interface{}) {
	for k, v := range headers {
		rw.Header().Add(k, string(v.([]byte)))
	}
}

// Flush - Sends output to client.
func Flush(rw http.ResponseWriter) {
	if fl, ok := rw.(http.Flusher); ok {
		fl.Flush()
	}
}

// WriteBody - Sends the body to the client.
func WriteBody(rw http.ResponseWriter, page string) bool {
	pageByte := []byte(page)
	sent, err := rw.Write(pageByte)

	// try again
	if sent == 0 && err != nil {
		log.Printf("Failed to Write: %s (Trying again)\n", err)

		sent, err = rw.Write(pageByte)
		return sent > 0 && err == nil
	}

	return true
}
