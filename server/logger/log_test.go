// +build unit

package logger_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

func TestLogMessage(t *testing.T) {
	setUpLog()

	reqMock := &http.Request{
		Proto:      "HTTPS",
		Method:     "POST",
		RemoteAddr: "127.0.0.1",
		URL:        &url.URL{Path: "/path/to/file"},
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	logger.Log(reqMock, "message")

	timeNow := time.Now().Local().Format("2006/01/02 15:04:05")
	expectedOut := fmt.Sprintf(`time="%s" level=info msg="HTTPS POST /path/to/file - message"`+"\n", timeNow)

	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func TestLogRequest(t *testing.T) {
	setUpLog()

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
	expectedOut := fmt.Sprintf(
		`time="%s" level=info msg="127.0.0.1 - - ? ? \"/path/to/file\" 404 7 \"https://www.google.com\" \"GoProxyCache\" true"`+"\n",
		timeNow,
	)

	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func TestLogSetup(t *testing.T) {
	setUpLog()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	config.Config = config.Configuration{
		Server: config.Server{
			Port: config.Port{
				HTTP:  "80",
				HTTPS: "443",
			},
			Forwarding: config.Forward{
				Host:      "www.google.com",
				Scheme:    "https",
				Endpoints: []string{"1.2.3.4", "8.8.8.8"},
			},
		},
	}

	logger.LogSetup(config.Config.Server)

	timeNow := time.Now().Local().Format("2006/01/02 15:04:05")

	expectedOut := fmt.Sprintf(
		`time="%s" level=info msg="Server will run on: 80 and 443\n"`+"\n"+
			`time="%s" level=info msg="Redirecting to url: https://www.google.com -> [1.2.3.4 8.8.8.8]\n"`+"\n",
		timeNow,
		timeNow,
	)
	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func setUpLog() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func tearDownLog() {
	config.Config = config.Configuration{}
}
