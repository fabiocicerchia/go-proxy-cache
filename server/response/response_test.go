// +build unit

package response_test

import (
	"net/http"
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/stretchr/testify/assert"
)

var MockStatusCode int
var MockContent []byte

type ResponseWriterMock struct {
	http.ResponseWriter
}

func (rwm ResponseWriterMock) WriteHeader(statusCode int) { MockStatusCode = statusCode }
func (rwm ResponseWriterMock) Write(p []byte) (int, error) {
	MockContent = p
	return 0, nil
}

func TestNewWriter(t *testing.T) {
	var rwMock ResponseWriterMock

	lwr := response.NewLoggedResponseWriter(rwMock)

	assert.Equal(t, 0, lwr.StatusCode)
	assert.Equal(t, "", string(lwr.Content))

	tearDownResponse()
}

func TestCatchStatusCode(t *testing.T) {
	var rwMock ResponseWriterMock

	lwr := response.NewLoggedResponseWriter(rwMock)
	lwr.WriteHeader(201)

	// checks lwr
	assert.Equal(t, 201, lwr.StatusCode)
	assert.Equal(t, "", string(lwr.Content))

	// verify calls on rwMock
	assert.Equal(t, 201, MockStatusCode)
	assert.Equal(t, "undefined", string(MockContent))

	tearDownResponse()
}

func TestCatchContent(t *testing.T) {
	var rwMock ResponseWriterMock

	lwr := response.NewLoggedResponseWriter(rwMock)

	content := []byte("test content")
	_, _ = lwr.Write(content)

	// checks lwr
	assert.Equal(t, 0, lwr.StatusCode)
	assert.Equal(t, content, lwr.Content)

	// verify calls on rwMock
	assert.Equal(t, -1, MockStatusCode)
	assert.Equal(t, content, MockContent)

	tearDownResponse()
}

func tearDownResponse() {
	MockStatusCode = -1
	MockContent = []byte("undefined")
}
