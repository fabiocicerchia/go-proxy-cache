package logger

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"fmt"
	"io"
	"log/syslog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/evalphobia/logrus_sentry"
	"github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/cache"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

var logFileHandle *os.File
var logLevel logrus.Level = logrus.InfoLevel
var Logger *logrus.Logger

// InitLogs - Configures basic settings for logging with logrus.
func InitLogs(verboseFlag bool, logFile string) {
	log := GetGlobal()

	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})

	log.SetReportCaller(verboseFlag)
	log.SetLevel(logLevel)
	log.SetOutput(os.Stdout)

	if logFile != "" {
		logFileHandle = getLogFileWriter(logFile)
		log.SetOutput(io.MultiWriter(logFileHandle))
		logrus.RegisterExitHandler(closeLogFile)
	}
}

// SetDebugLevel - Changes log level to DEBUG.
func SetDebugLevel() {
	logLevel = logrus.DebugLevel
}

func getLogFileWriter(logFile string) *os.File {
	f, err := os.OpenFile(filepath.Clean(logFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		Logger.Fatal(err)
	}

	return f
}

func closeLogFile() {
	if logFileHandle != nil {
		_ = logFileHandle.Close()
	}
}

// Log - Logs against a requested URL.
func Log(req http.Request, reqID string, message string) {
	escapedMessage := strings.Replace(message, "\n", "", -1)
	escapedMessage = strings.Replace(escapedMessage, "\r", "", -1)
	escapedURL := strings.Replace(req.URL.String(), "\n", "", -1)
	escapedURL = strings.Replace(escapedURL, "\r", "", -1)

	log := GetGlobal()
	log.WithFields(logrus.Fields{"ReqID": reqID}).Infof("%s %s %s - %s", req.Proto, req.Method, escapedURL, escapedMessage)
}

// LogRequest - Logs the requested URL.
func LogRequest(req http.Request, statusCode int, lenContent int, reqID string, cacheLabel int) {
	cached := cacheLabel != cache.StatusMiss
	cachedLabel := cache.StatusLabel[cacheLabel]

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
		`$status`, strconv.Itoa(statusCode),
		`$body_bytes_sent`, strconv.Itoa(lenContent),
		`$http_referer`, req.Referer(),
		`$http_user_agent`, req.UserAgent(),
		`$cached_status_label`, cachedLabel,
		`$cached_status`, fmt.Sprintf("%v", cached),
	)

	logLine = r.Replace(logLine)

	log := GetGlobal()
	log.WithFields(logrus.Fields{"ReqID": reqID}).Info(logLine)
}

// LogSetup - Logs the env variables required for a reverse proxy.
func LogSetup(server config.Server) {
	forwardHost := utils.IfEmpty(server.Upstream.Host, "*")
	forwardProto := server.Upstream.Scheme

	lbEndpointList := fmt.Sprintf("%v", server.Upstream.Endpoints)
	if len(server.Upstream.Endpoints) == 0 {
		lbEndpointList = "VOID"
	}

	log := GetGlobal()
	log.Infof("Server will run on :%s and :%s and redirects to url: %s://%s -> %s\n", server.Port.HTTP, server.Port.HTTPS, forwardProto, forwardHost, lbEndpointList)
}

// GetGlobal - Returns existing instance of global logger (it'll create a new one if doesn't exist).
func GetGlobal() *logrus.Logger {
	if Logger == nil {
		Logger = logrus.New()
	}

	return Logger
}

// HookSentry - Configures (optionally) the Sentry hook for logrus.
func HookSentry(log *logrus.Logger, sentryDsn string) {
	if sentryDsn == "" {
		return
	}

	hook, err := logrus_sentry.NewSentryHook(sentryDsn, []logrus.Level{ // TODO: Make them customizable?
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
	})

	if err != nil {
		panic(err)
	}

	hook.StacktraceConfiguration.Enable = true
	log.Hooks.Add(hook)
}

// HookSyslog - Configures (optionally) the syslog hook for logrus.
func HookSyslog(log *logrus.Logger, syslogProtocol string, syslogEndpoint string) {
	if syslogEndpoint == "" {
		return
	}

	hook, err := lSyslog.NewSyslogHook(syslogProtocol, syslogEndpoint, syslog.LOG_WARNING, "")

	if err != nil {
		panic(err)
	}

	log.Hooks.Add(hook)
}
