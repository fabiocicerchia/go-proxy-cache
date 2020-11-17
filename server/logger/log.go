package logger

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

const logFormat = `$remote_addr - $remote_user "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status`

// https://golang.org/src/time/format.go
const dateFormat = "2006/01/02 15:04:05"

// LogRequest - Logs the requested URL.
func LogRequest(req *http.Request, res *response.LoggedResponseWriter, cached bool) {
	logLine := logFormat

	// TODO: replace with array mapping
	logLine = strings.ReplaceAll(logLine, `$remote_addr`, req.RemoteAddr)
	logLine = strings.ReplaceAll(logLine, `$remote_user`, "-")
	logLine = strings.ReplaceAll(logLine, `$time_local`, time.Now().Local().Format(dateFormat))
	logLine = strings.ReplaceAll(logLine, `$request`, req.URL.String())
	logLine = strings.ReplaceAll(logLine, `$status`, strconv.Itoa(res.StatusCode))
	logLine = strings.ReplaceAll(logLine, `$body_bytes_sent`, strconv.Itoa(len(res.Content)))
	logLine = strings.ReplaceAll(logLine, `$http_referer`, req.Referer())
	logLine = strings.ReplaceAll(logLine, `$http_user_agent`, req.UserAgent())
	logLine = strings.ReplaceAll(logLine, `$cached_status`, fmt.Sprintf("%v", cached))

	log.Println(logLine)
}

// LogSetup - Logs the env variables required for a reverse proxy.
func LogSetup(forwarding config.Forward, port string) {
	forwardHost := forwarding.Host
	forwardProto := forwarding.Scheme
	lbEndpointList := forwarding.Endpoints

	log.Printf("Server will run on: %s\n", port)
	log.Printf("Redirecting to url: %s://%s -> %v\n", forwardProto, forwardHost, lbEndpointList)
}
