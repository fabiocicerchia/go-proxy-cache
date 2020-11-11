package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// const logFormat = `$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"`
const logFormat = `$remote_addr - $remote_user "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"`

// https://golang.org/src/time/format.go
const dateFormat = "2006/01/02 15:04:05"

// Log the redirect url
func logRequest(proxyURL string, req *http.Request) {
	log_line := logFormat
	log_line = strings.ReplaceAll(log_line, `$remote_addr`, req.RemoteAddr)
	log_line = strings.ReplaceAll(log_line, `$remote_user`, "-")
	log_line = strings.ReplaceAll(log_line, `$time_local`, time.Now().Local().Format(dateFormat))
	log_line = strings.ReplaceAll(log_line, `$request`, proxyURL+req.URL.String())
	// log_line = strings.ReplaceAll(log_line, `$status`, "")
	// log_line = strings.ReplaceAll(log_line, `$body_bytes_sent`, "")
	log_line = strings.ReplaceAll(log_line, `$http_referer`, req.Header.Get("Referer"))
	log_line = strings.ReplaceAll(log_line, `$http_user_agent`, req.Header.Get("User-Agent"))

	log.Println(log_line)
}

// Log the env variables required for a reverse proxy
func logSetup() {
	forward_to := utils.GetEnv("FORWARD_TO", "")

	log.Printf("Server will run on: %s\n", getListenAddress())
	log.Printf("Redirecting to url: %s\n", forward_to)
}
