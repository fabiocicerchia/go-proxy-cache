package logger_test

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

func TestLogRequest(t *testing.T) {
	reqMock := &http.Request{
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
	}
	reqMock.Header = make(http.Header)
	reqMock.Header.Set("Referer", "https://www.google.com")
	reqMock.Header.Set("User-Agent", "GoProxyCache")

	lwrMock := &response.LoggedResponseWriter{
		StatusCode: 404,
		Content:    []byte("testing"),
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	logger.LogRequest(reqMock, lwrMock, true)

	timeNow := time.Now().Local().Format("2006/01/02 15:04:05")
	expectedOut := fmt.Sprintf(`%s 127.0.0.1 - - "/path/to/file" 404 7 "https://www.google.com" "GoProxyCache" true`+"\n", timeNow)

	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func TestLogSetup(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	config.Config = config.Configuration{
		Server: config.Server{
			Forwarding: config.Forward{
				Host:      "www.google.com",
				Scheme:    "https",
				Endpoints: []string{"1.2.3.4", "8.8.8.8"},
			},
		},
	}

	logger.LogSetup(config.Config.Server.Forwarding, "8081")

	timeNow := time.Now().Local().Format("2006/01/02 15:04:05")

	expectedOut := fmt.Sprintf("%s Server will run on: 8081\n%s Redirecting to url: https://www.google.com -> [1.2.3.4 8.8.8.8]\n", timeNow, timeNow)
	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func tearDownLog() {
	config.Config = config.Configuration{}
}
