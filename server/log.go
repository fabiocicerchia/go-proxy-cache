package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

const logFormat = `$remote_addr - $remote_user "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status`

// https://golang.org/src/time/format.go
const dateFormat = "2006/01/02 15:04:05"

// Log the redirect url
func LogRequest(req *http.Request, res *LoggedResponseWriter, cached bool) {
	log_line := logFormat

	log_line = strings.ReplaceAll(log_line, `$remote_addr`, req.RemoteAddr)
	log_line = strings.ReplaceAll(log_line, `$remote_user`, "-")
	log_line = strings.ReplaceAll(log_line, `$time_local`, time.Now().Local().Format(dateFormat))
	log_line = strings.ReplaceAll(log_line, `$request`, req.URL.String())
	log_line = strings.ReplaceAll(log_line, `$status`, strconv.Itoa(res.StatusCode))
	log_line = strings.ReplaceAll(log_line, `$body_bytes_sent`, strconv.Itoa(len(res.Content)))
	log_line = strings.ReplaceAll(log_line, `$http_referer`, req.Referer())
	log_line = strings.ReplaceAll(log_line, `$http_user_agent`, req.UserAgent())
	log_line = strings.ReplaceAll(log_line, `$cached_status`, fmt.Sprintf("%v", cached))

	log.Println(log_line)
}

// Log the env variables required for a reverse proxy
func LogSetup(port string) {
	forwardHost := config.Config.Server.Forwarding.Host
	forwardProto := config.Config.Server.Forwarding.Scheme
	lbEndpointList := config.Config.Server.Forwarding.Endpoints

	log.Printf("Server will run on: %s\n", port)
	log.Printf("Redirecting to url: %s://%s -> %v\n", forwardProto, forwardHost, lbEndpointList)
}
