package logger

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

const logFormat = `$remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status`

// https://golang.org/src/time/format.go
const dateFormat = "2006/01/02 15:04:05"

// Log - Logs against a requested URL.
func Log(req *http.Request, message string) {
	logLine := fmt.Sprintf("%s %s %s - %s", req.Proto, req.Method, req.URL.String(), message)

	log.Info(logLine)
}

// LogRequest - Logs the requested URL.
func LogRequest(req *http.Request, res *response.LoggedResponseWriter, cached bool) {
	logLine := logFormat

	protocol := strings.Trim(req.Proto, " ")
	if protocol == "" {
		protocol = "?"
	}

	method := strings.Trim(req.Method, " ")
	if method == "" {
		method = "?"
	}

	// TODO: replace with array mapping
	logLine = strings.ReplaceAll(logLine, `$remote_addr`, req.RemoteAddr)
	logLine = strings.ReplaceAll(logLine, `$remote_user`, "-")
	logLine = strings.ReplaceAll(logLine, `$time_local`, time.Now().Local().Format(dateFormat))
	logLine = strings.ReplaceAll(logLine, `$protocol`, protocol)
	logLine = strings.ReplaceAll(logLine, `$request_method`, method)
	logLine = strings.ReplaceAll(logLine, `$request`, req.URL.String())
	logLine = strings.ReplaceAll(logLine, `$status`, strconv.Itoa(res.StatusCode))
	logLine = strings.ReplaceAll(logLine, `$body_bytes_sent`, strconv.Itoa(len(res.Content)))
	logLine = strings.ReplaceAll(logLine, `$http_referer`, req.Referer())
	logLine = strings.ReplaceAll(logLine, `$http_user_agent`, req.UserAgent())
	logLine = strings.ReplaceAll(logLine, `$cached_status`, fmt.Sprintf("%v", cached))

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
