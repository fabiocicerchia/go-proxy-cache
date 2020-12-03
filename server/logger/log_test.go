// +build all unit

package logger_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

func TestLogMessage(t *testing.T) {
	setUpLog()

	reqMock := http.Request{
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

	config.Config = config.Configuration{
		Log: config.Log{
			TimeFormat: "2006/01/02 15:04:05",
			Format:     `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status`,
		},
	}

	logger.Log(reqMock, "message")

	expectedOut := `time=" " level=info msg="HTTPS POST /path/to/file - message"` + "\n"

	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func TestLogRequest(t *testing.T) {
	setUpLog()

	reqMock := http.Request{
		RemoteAddr: "127.0.0.1",
		Host:       "example.org",
		URL:        &url.URL{Path: "/path/to/file"},
	}
	reqMock.Header = make(http.Header)
	reqMock.Header.Set("Referer", "https://www.google.com")
	reqMock.Header.Set("User-Agent", "GoProxyCache")

	lwrMock := response.LoggedResponseWriter{
		StatusCode: 404,
		Content:    [][]byte{[]byte("testing")},
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	config.Config = config.Configuration{
		Log: config.Log{
			TimeFormat: "2006/01/02 15:04:05",
			Format:     `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status`,
		},
	}

	logger.LogRequest(reqMock, lwrMock, true)

	expectedOut := `time=" " level=info msg="example.org - 127.0.0.1 - - ? ? \"/path/to/file\" 404 7 \"https://www.google.com\" \"GoProxyCache\" true"` + "\n"

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
		Log: config.Log{
			TimeFormat: "2006/01/02 15:04:05",
			Format:     `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status`,
		},
	}

	logger.LogSetup(config.Config.Server)

	expectedOut := `time=" " level=info msg="Server will run on: 80 and 443\n"` + "\n" +
		`time=" " level=info msg="Redirecting to url: https://www.google.com -> [1.2.3.4 8.8.8.8]\n"` + "\n"
	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func setUpLog() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   false,
		TimestampFormat: " ",
	})
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func tearDownLog() {
	config.Config = config.Configuration{}
}
