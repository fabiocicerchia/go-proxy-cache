package server

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
)

func TestLogRequest(t *testing.T) {
	reqMock := &http.Request{
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
	}
	reqMock.Header = make(http.Header)
	reqMock.Header.Set("Referer", "https://www.google.com")
	reqMock.Header.Set("User-Agent", "GoProxyCache")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	logRequest("https://www.example.com", reqMock)

	timeNow := time.Now().Local().Format("2006/01/02 15:04:05")
	expectedOut := fmt.Sprintf(`%s 127.0.0.1 - - "https://www.example.com/path/to/file" $status $body_bytes_sent "https://www.google.com" "GoProxyCache"`+"\n", timeNow)

	assert.Equal(t, expectedOut, buf.String())
}

func TestLogSetup(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	os.Setenv("FORWARD_TO", "https://www.google.com")

	logSetup()

	timeNow := time.Now().Local().Format("2006/01/02 15:04:05")

	expectedOut := fmt.Sprintf("%s Server will run on: :8080\n%s Redirecting to url: https://www.google.com\n", timeNow, timeNow)
	assert.Equal(t, expectedOut, buf.String())

	tearDown()
}
