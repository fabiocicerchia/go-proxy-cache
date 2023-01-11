//go:build all || unit
// +build all unit

package logger_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
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
	"github.com/fabiocicerchia/go-proxy-cache/logger"
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
	logger.Logger.SetOutput(&buf)
	defer func() {
		logger.Logger.SetOutput(os.Stderr)
	}()

	config.Config = config.Configuration{
		Log: config.Log{
			TimeFormat: "2006/01/02 15:04:05",
			Format:     `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status $cached_status_label`,
		},
	}

	logger.Log(reqMock, "TestLogMessage", "message")

	expectedOut := `time=" " level=info msg="HTTPS POST /path/to/file - message" ReqID=TestLogMessage` + "\n"

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
	logger.Logger.SetOutput(&buf)
	defer func() {
		logger.Logger.SetOutput(os.Stderr)
	}()

	config.Config = config.Configuration{
		Log: config.Log{
			TimeFormat: "2006/01/02 15:04:05",
			Format:     `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status $cached_status_label`,
		},
	}

	logger.LogRequest(reqMock, lwrMock.StatusCode, lwrMock.Content.Len(), "TestLogRequest", 1)

	expectedOut := `time=" " level=info msg="example.org - 127.0.0.1 - - ? ? \"/path/to/file\" 404 7 \"https://www.google.com\" \"GoProxyCache\" true HIT" ReqID=TestLogRequest` + "\n"

	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func TestLogSetup(t *testing.T) {
	setUpLog()

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)
	defer func() {
		logger.Logger.SetOutput(os.Stderr)
	}()

	cfg := config.Configuration{
		Server: config.Server{
			Port: config.Port{
				HTTP:  "80",
				HTTPS: "443",
			},
			Upstream: config.Upstream{
				Host:      "www.google.com",
				Scheme:    "https",
				Endpoints: []string{"1.2.3.4", "8.8.8.8"},
			},
		},
	}
	config.Config = config.Configuration{
		Log: config.Log{
			TimeFormat: "2006/01/02 15:04:05",
			Format:     `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status $cached_status_label`,
		},
	}

	logger.LogSetup(cfg.Server)

	expectedOut := `time=" " level=info msg="Server will run on :80 and :443 and redirects to url: https://www.google.com -> [1.2.3.4 8.8.8.8]\n"` + "\n"
	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func TestLogSetupWithoutEndpoints(t *testing.T) {
	setUpLog()

	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)
	defer func() {
		logger.Logger.SetOutput(os.Stderr)
	}()

	cfg := config.Configuration{
		Server: config.Server{
			Port: config.Port{
				HTTP:  "80",
				HTTPS: "443",
			},
			Upstream: config.Upstream{
				Host:      "www.google.com",
				Scheme:    "https",
				Endpoints: []string{},
			},
		},
	}
	config.Config = config.Configuration{
		Log: config.Log{
			TimeFormat: "2006/01/02 15:04:05",
			Format:     `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status $cached_status_label`,
		},
	}

	logger.LogSetup(cfg.Server)

	expectedOut := `time=" " level=info msg="Server will run on :80 and :443 and redirects to url: https://www.google.com -> VOID\n"` + "\n"
	assert.Equal(t, expectedOut, buf.String())

	tearDownLog()
}

func setUpLog() {
	logger.Logger = logger.GetGlobal()

	logger.Logger.SetFormatter(&log.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   false,
		TimestampFormat: " ",
	})
	logger.Logger.SetReportCaller(false)
	logger.Logger.SetOutput(os.Stdout)
	logger.Logger.SetLevel(log.InfoLevel)
}

func tearDownLog() {
	config.Config = config.Configuration{}
}
