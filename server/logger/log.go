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
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

// Log - Logs against a requested URL.
func Log(req http.Request, message string) {
	logLine := fmt.Sprintf("%s %s %s - %s", req.Proto, req.Method, req.URL.String(), message)

	log.Info(logLine)
}

// LogRequest - Logs the requested URL.
func LogRequest(req http.Request, lwr response.LoggedResponseWriter, cached bool) {
	logLine := config.Config.Log.Format

	protocol := strings.Trim(req.Proto, " ")
	if protocol == "" {
		protocol = "?"
	}

	method := strings.Trim(req.Method, " ")
	if method == "" {
		method = "?"
	}

	r := strings.NewReplacer(
		`$host`, req.Host,
		`$remote_addr`, req.RemoteAddr,
		`$remote_user`, "-",
		`$time_local`, time.Now().Local().Format(config.Config.Log.TimeFormat),
		`$protocol`, protocol,
		`$request_method`, method,
		`$request`, req.URL.String(),
		`$status`, strconv.Itoa(lwr.StatusCode),
		`$body_bytes_sent`, strconv.Itoa(slice.LenSliceBytes(lwr.Content)),
		`$http_referer`, req.Referer(),
		`$http_user_agent`, req.UserAgent(),
		`$cached_status`, fmt.Sprintf("%v", cached),
	)

	logLine = r.Replace(logLine)

	log.Info(logLine)
}

// LogSetup - Logs the env variables required for a reverse proxy.
func LogSetup(server config.Server) {
	forwardHost := server.Upstream.Host
	forwardProto := utils.IfEmpty(server.Upstream.Scheme, "*")
	lbEndpointList := server.Upstream.Endpoints

	log.Infof("Server will run on: %s and %s\n", server.Port.HTTP, server.Port.HTTPS)
	log.Infof("Redirecting to url: %s://%s -> %v\n", forwardProto, forwardHost, lbEndpointList)
}
