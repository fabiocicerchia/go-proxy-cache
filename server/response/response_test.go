//go:build all || unit
// +build all unit

package response_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

var MockStatusCode int
var MockContent response.DataChunks

type ResponseWriterMock struct {
	http.ResponseWriter
}

func (rwm ResponseWriterMock) WriteHeader(statusCode int) {
	MockStatusCode = statusCode
}
func (rwm ResponseWriterMock) Write(p []byte) (int, error) {
	MockContent = append(MockContent, []byte{})
	chunk := len(MockContent) - 1
	MockContent[chunk] = append(MockContent[chunk], p...)
	return 0, nil
}

func initLogs() {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
}

func TestNewWriter(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestNewWriter")

	assert.Equal(t, 0, lwr.StatusCode)
	assert.Len(t, lwr.Content, 0)

	tearDownResponse()
}

func TestCatchStatusCode(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestCatchStatusCode")
	lwr.WriteHeader(http.StatusCreated)

	// checks lwr
	assert.Equal(t, http.StatusCreated, lwr.StatusCode)
	assert.Len(t, lwr.Content, 0)

	// verify calls on rwMock
	assert.Equal(t, -1, MockStatusCode)
	assert.Len(t, MockContent, 0)

	tearDownResponse()
}

func TestCatchStatusCodeForced(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestCatchStatusCodeForced")
	lwr.ForceWriteHeader(http.StatusCreated)

	// checks lwr
	assert.Equal(t, http.StatusCreated, lwr.StatusCode)
	assert.Len(t, lwr.Content, 0)

	// verify calls on rwMock
	assert.Equal(t, http.StatusCreated, MockStatusCode)
	assert.Len(t, MockContent, 0)

	tearDownResponse()
}

func TestCatchContent(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestCatchContent")

	content := []byte("test content")
	_, err := lwr.Write(content)
	assert.Nil(t, err)

	expectedContent := response.DataChunks{content}

	// checks lwr
	// even if don't set it explicitly, it fallback on 200
	assert.Equal(t, http.StatusOK, lwr.StatusCode)
	assert.Equal(t, expectedContent, lwr.Content)

	// verify calls on rwMock
	assert.Equal(t, -1, MockStatusCode)
	// Empty because buffered.
	assert.Equal(t, 0, MockContent.Len())
	assert.Equal(t, []byte{}, MockContent.Bytes())
	assert.Equal(t, response.DataChunks{}, MockContent)

	tearDownResponse()
}

func TestCatchContentForced(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestCatchContentForced")

	content := []byte("test content")
	_, err := lwr.ForceWrite(content)
	assert.Nil(t, err)

	expectedContent := response.DataChunks{content}

	// checks lwr
	// even if don't set it explicitly, it fallback on 200
	assert.Equal(t, http.StatusOK, lwr.StatusCode)
	assert.Equal(t, expectedContent, lwr.Content)

	// verify calls on rwMock
	assert.Equal(t, -1, MockStatusCode)
	assert.Equal(t, expectedContent, MockContent)
	assert.Equal(t, 12, MockContent.Len())
	assert.Equal(t, content, MockContent.Bytes())

	tearDownResponse()
}

func TestCatchContentThreeChunks(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestCatchContentThreeChunks")

	content := []byte("test content")
	content2 := []byte("test content2")
	content3 := []byte("test content3")
	_, err := lwr.ForceWrite(content)
	assert.Nil(t, err)
	_, err = lwr.ForceWrite(content2)
	assert.Nil(t, err)
	_, err = lwr.ForceWrite(content3)
	assert.Nil(t, err)

	expectedContent := response.DataChunks{content, content2, content3}

	// checks lwr
	// even if don't set it explicitly, it fallback on 200
	assert.Equal(t, http.StatusOK, lwr.StatusCode)
	assert.Equal(t, expectedContent, lwr.Content)

	// verify calls on rwMock
	assert.Equal(t, -1, MockStatusCode)
	assert.Equal(t, expectedContent, MockContent)
	assert.Equal(t, 38, MockContent.Len())

	tearDownResponse()
}

func TestSendNotImplemented(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestSendNotImplemented")
	lwr.SendNotImplemented()

	// checks lwr
	assert.Equal(t, http.StatusNotImplemented, lwr.StatusCode)

	// verify calls on rwMock
	assert.Equal(t, http.StatusNotImplemented, MockStatusCode)

	tearDownResponse()
}

func TestSendNotModifiedResponse(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestSendNotModifiedResponse")
	lwr.SendNotModifiedResponse()

	// checks lwr
	// it sends only to the internal writer
	assert.Equal(t, 0, lwr.StatusCode)
	assert.Equal(t, response.DataChunks{}, lwr.Content)

	// verify calls on rwMock
	assert.Equal(t, http.StatusNotModified, MockStatusCode)
	assert.Equal(t, response.DataChunks{[]byte{}}, MockContent)

	tearDownResponse()
}

func TestGetETagWeak(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestGetETagWeak")

	etag := lwr.GetETag(true)

	assert.Regexp(t, regexp.MustCompile(`^\"W/[0-9]+-[0-9a-f]{64}\"$`), etag)
}

func TestGetETagNotWeak(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestGetETagNotWeak")

	etag := lwr.GetETag(false)

	assert.Regexp(t, regexp.MustCompile(`^\"[0-9]+-[0-9a-f]{64}\"$`), etag)
}

func TestSetETagWeak(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{ResponseWriter: httptest.NewRecorder()}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestSetETagWeak")
	lwr.SetETag(true)

	assert.Regexp(t, regexp.MustCompile(`^\"W/[0-9]+-[0-9a-f]{64}\"$`), lwr.ResponseWriter.Header().Get("ETag"))
}

func TestSetETagNotWeak(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{ResponseWriter: httptest.NewRecorder()}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestSetETagNotWeak")
	lwr.SetETag(false)

	assert.Regexp(t, regexp.MustCompile(`^\"[0-9]+-[0-9a-f]{64}\"$`), lwr.ResponseWriter.Header().Get("ETag"))
}

func TestInitGZipBuffer(t *testing.T) {
	initLogs()

	rwMock := ResponseWriterMock{ResponseWriter: httptest.NewRecorder()}

	lwr := response.NewLoggedResponseWriter(rwMock, "TestInitGZipBuffer")
	lwr.InitGZipBuffer()

	assert.NotNil(t, lwr.GZipResponse)
	assert.IsType(t, &gzip.Writer{}, lwr.GZipResponse)
}

func tearDownResponse() {
	MockStatusCode = -1
	MockContent = make(response.DataChunks, 0)
}
