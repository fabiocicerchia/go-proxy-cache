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

// Log - Logs against a requested URL.
func Log(req http.Request, reqID string, message string) {
	logLine := fmt.Sprintf("%s %s %s - %s", req.Proto, req.Method, req.URL.String(), message)

	log.WithFields(log.Fields{"ReqID": reqID}).Info(logLine)
}

// LogRequest - Logs the requested URL.
func LogRequest(req http.Request, lwr response.LoggedResponseWriter, reqID string, cached bool, cached_label string) {
	// NOTE: THIS IS FOR EVERY DOMAIN, NO DOMAIN OVERRIDE.
	//       WHEN SHARING SAME PORT NO CUSTOM OVERRIDES ON CRITICAL SETTINGS.
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
		`$body_bytes_sent`, strconv.Itoa(lwr.Content.Len()),
		`$http_referer`, req.Referer(),
		`$http_user_agent`, req.UserAgent(),
		`$cached_status_label`, cached_label,
		`$cached_status`, fmt.Sprintf("%v", cached),
	)

	logLine = r.Replace(logLine)

	log.WithFields(log.Fields{"ReqID": reqID}).Info(logLine)
}

// LogSetup - Logs the env variables required for a reverse proxy.
func LogSetup(server config.Server) {
	forwardHost := utils.IfEmpty(server.Upstream.Host, "*")
	forwardProto := server.Upstream.Scheme

	lbEndpointList := fmt.Sprintf("%v", server.Upstream.Endpoints)
	if len(lbEndpointList) == 0 {
		lbEndpointList = "VOID" // TODO: COVER
	}

	log.Infof("Server will run on :%s and :%s and redirects to url: %s://%s -> %s\n", server.Port.HTTP, server.Port.HTTPS, forwardProto, forwardHost, lbEndpointList)
}
