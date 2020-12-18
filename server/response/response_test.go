// +build all unit

package response_test

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
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/stretchr/testify/assert"
)

var MockStatusCode int
var MockContent [][]byte

type ResponseWriterMock struct {
	http.ResponseWriter
}

func (rwm ResponseWriterMock) WriteHeader(statusCode int) { MockStatusCode = statusCode }
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

	var rwMock ResponseWriterMock

	lwr := response.NewLoggedResponseWriter(rwMock)

	assert.Equal(t, 0, lwr.StatusCode)
	assert.Len(t, lwr.Content, 0)

	tearDownResponse()
}

func TestCatchStatusCode(t *testing.T) {
	initLogs()

	var rwMock ResponseWriterMock

	lwr := response.NewLoggedResponseWriter(rwMock)
	lwr.WriteHeader(http.StatusCreated)

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

	var rwMock ResponseWriterMock

	lwr := response.NewLoggedResponseWriter(rwMock)

	content := []byte("test content")
	_, err := lwr.Write(content)
	assert.Nil(t, err)

	expectedContent := [][]byte{content}

	// checks lwr
	assert.Equal(t, 0, lwr.StatusCode)
	assert.Equal(t, expectedContent, lwr.Content)

	// verify calls on rwMock
	assert.Equal(t, -1, MockStatusCode)
	assert.Equal(t, expectedContent, MockContent)

	tearDownResponse()
}

func TestCatchContentThreeChunks(t *testing.T) {
	initLogs()

	var rwMock ResponseWriterMock

	lwr := response.NewLoggedResponseWriter(rwMock)

	content := []byte("test content")
	content2 := []byte("test content2")
	content3 := []byte("test content3")
	_, err := lwr.Write(content)
	assert.Nil(t, err)
	_, err = lwr.Write(content2)
	assert.Nil(t, err)
	_, err = lwr.Write(content3)
	assert.Nil(t, err)

	expectedContent := [][]byte{content, content2, content3}

	// checks lwr
	assert.Equal(t, 0, lwr.StatusCode)
	assert.Equal(t, expectedContent, lwr.Content)

	// verify calls on rwMock
	assert.Equal(t, -1, MockStatusCode)
	assert.Equal(t, expectedContent, MockContent)

	tearDownResponse()
}

func tearDownResponse() {
	MockStatusCode = -1
	MockContent = make([][]byte, 0)
}
