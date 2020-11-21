package response

import (
	"net/http"

	log "github.com/sirupsen/logrus"
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
	// TODO: COVERAGE: need to find a different domain in config.yml (at it will fix itself)
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

	// try again
	if sent == 0 && err != nil {
		log.Warnf("Failed to Write: %s (Trying again)\n", err)

		sent, err = rw.Write(pageByte)
		return sent > 0 && err == nil
	}

	return true
}
