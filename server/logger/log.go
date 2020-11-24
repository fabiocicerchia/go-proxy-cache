package logger

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

const logFormat = `$remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status`

// https://golang.org/src/time/format.go
const dateFormat = "2006/01/02 15:04:05"

// Log - Logs against a requested URL.
func Log(req http.Request, message string) {
	logLine := fmt.Sprintf("%s %s %s - %s", req.Proto, req.Method, req.URL.String(), message)

	log.Info(logLine)
}

// LogRequest - Logs the requested URL.
func LogRequest(req http.Request, lwr response.LoggedResponseWriter, cached bool) {
	logLine := logFormat

	protocol := strings.Trim(req.Proto, " ")
	if protocol == "" {
		protocol = "?"
	}

	method := strings.Trim(req.Method, " ")
	if method == "" {
		method = "?"
	}

	r := strings.NewReplacer(
		`$remote_addr`, req.RemoteAddr,
		`$remote_user`, "-",
		`$time_local`, time.Now().Local().Format(dateFormat),
		`$protocol`, protocol,
		`$request_method`, method,
		`$request`, req.URL.String(),
		`$status`, strconv.Itoa(lwr.StatusCode),
		`$body_bytes_sent`, strconv.Itoa(utils.LenSliceBytes(lwr.Content)),
		`$http_referer`, req.Referer(),
		`$http_user_agent`, req.UserAgent(),
		`$cached_status`, fmt.Sprintf("%v", cached),
	)

	logLine = r.Replace(logLine)

	log.Info(logLine)
}

// LogSetup - Logs the env variables required for a reverse proxy.
func LogSetup(server config.Server) {
	forwardHost := server.Forwarding.Host
	forwardProto := server.Forwarding.Scheme
	lbEndpointList := server.Forwarding.Endpoints

	log.Infof("Server will run on: %s and %s\n", server.Port.HTTP, server.Port.HTTPS)
	log.Infof("Redirecting to url: %s://%s -> %v\n", forwardProto, forwardHost, lbEndpointList)
}
