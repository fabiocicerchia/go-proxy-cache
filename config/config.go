package config

import (
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	log "github.com/sirupsen/logrus"
)

// Config - Holds the server configuration
var Config Configuration

// Configuration - Defines the server configuration
type Configuration struct {
	Server Server
	Cache  Cache
}

// Server - Defines basic info for the server
type Server struct {
	Port       Port
	TLS        TLS
	Timeout    Timeout
	Forwarding Forward
}

// Port - Defines the listening ports per protocol
type Port struct {
	HTTP  string
	HTTPS string
}

// TLS - Defines the configuration for SSL/TLS
type TLS struct {
	Auto     bool
	Email    string
	CertFile string
	KeyFile  string
}

// Forward - Defines the forwarding settings
type Forward struct {
	Host               string
	Scheme             string
	Endpoints          []string
	HTTP2HTTPS         bool
	RedirectStatusCode int
}

// Timeout - Defines the server timeouts
type Timeout struct {
	Read       time.Duration
	ReadHeader time.Duration
	Write      time.Duration
	Idle       time.Duration
	Handler    time.Duration
}

// Cache - Defines the config for the cache backend
type Cache struct {
	Host            string
	Port            string
	Password        string
	DB              int
	TTL             int
	AllowedStatuses []string
	AllowedMethods  []string
}

// Coalesce - Returns the original value if the conditions is not met, fallback value otherwise.
func Coalesce(value interface{}, fallback interface{}, condition bool) interface{} {
	if condition {
		value = fallback
	}

	return value
}

// InitConfigFromFileOrEnv - Init the configuration in sequence: from a YAML file, from environment variables,
// then defaults.
func InitConfigFromFileOrEnv(file string) {
	Config = Configuration{}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Warnf("Cannot read file %s: %s\n", file, err)
	}
	err = yaml.Unmarshal([]byte(data), &Config)
	if err != nil {
		log.Warnf("Cannot unmarshal yaml: %s\n", err)
	}

	// --- Server

	Config.Server.Port.HTTP = Coalesce(Config.Server.Port.HTTP, utils.GetEnv("SERVER_HTTP_PORT", "80"), Config.Server.Port.HTTP == "").(string)
	Config.Server.Port.HTTPS = Coalesce(Config.Server.Port.HTTPS, utils.GetEnv("SERVER_HTTPS_PORT", "443"), Config.Server.Port.HTTPS == "").(string)

	// --- TLS

	autoTLSCertVal, err := strconv.Atoi(utils.GetEnv("TLS_AUTO_CERT", "0"))
	autoTLSCertVal = Coalesce(autoTLSCertVal, 0, err != nil).(int)
	autoTLSCert := autoTLSCertVal == 1

	Config.Server.TLS.Auto = Coalesce(Config.Server.TLS.Auto, autoTLSCert, !Config.Server.TLS.Auto).(bool)
	Config.Server.TLS.Email = Coalesce(Config.Server.TLS.Email, utils.GetEnv("TLS_EMAIL", ""), Config.Server.TLS.Email == "").(string)
	Config.Server.TLS.CertFile = Coalesce(Config.Server.TLS.CertFile, utils.GetEnv("TLS_CERT_FILE", ""), Config.Server.TLS.CertFile == "").(string)
	Config.Server.TLS.KeyFile = Coalesce(Config.Server.TLS.KeyFile, utils.GetEnv("TLS_KEY_FILE", ""), Config.Server.TLS.KeyFile == "").(string)

	// --- Timeout

	timeoutRead, err := strconv.Atoi(utils.GetEnv("TIMEOUT_READ", ""))
	timeoutReadTime := time.Duration(Coalesce(timeoutRead, 5000000000, timeoutRead == 0 || err != nil).(int))

	timeoutReadHeader, err := strconv.Atoi(utils.GetEnv("TIMEOUT_READ_HEADER", ""))
	timeoutReadHeaderTime := time.Duration(Coalesce(timeoutReadHeader, 2000000000, timeoutReadHeader == 0 || err != nil).(int))

	timeoutWrite, err := strconv.Atoi(utils.GetEnv("TIMEOUT_WRITE", ""))
	timeoutWriteTime := time.Duration(Coalesce(timeoutWrite, 5000000000, timeoutWrite == 0 || err != nil).(int))

	timeoutIdle, err := strconv.Atoi(utils.GetEnv("TIMEOUT_IDLE", ""))
	timeoutIdleTime := time.Duration(Coalesce(timeoutIdle, 20000000000, timeoutIdle == 0 || err != nil).(int))

	timeoutHandler, err := strconv.Atoi(utils.GetEnv("TIMEOUT_HANDLER", ""))
	timeoutHandlerTime := time.Duration(Coalesce(timeoutHandler, 5000000000, timeoutHandler == 0 || err != nil).(int))

	Config.Server.Timeout.Read = Coalesce(Config.Server.Timeout.Read, timeoutReadTime, Config.Server.Timeout.Read == 0).(time.Duration)
	Config.Server.Timeout.ReadHeader = Coalesce(Config.Server.Timeout.ReadHeader, timeoutReadHeaderTime, Config.Server.Timeout.ReadHeader == 0).(time.Duration)
	Config.Server.Timeout.Write = Coalesce(Config.Server.Timeout.Write, timeoutWriteTime, Config.Server.Timeout.Write == 0).(time.Duration)
	Config.Server.Timeout.Idle = Coalesce(Config.Server.Timeout.Idle, timeoutIdleTime, Config.Server.Timeout.Idle == 0).(time.Duration)
	Config.Server.Timeout.Handler = Coalesce(Config.Server.Timeout.Handler, timeoutHandlerTime, Config.Server.Timeout.Handler == 0).(time.Duration)

	// --- Forwarding

	lbEnpointList := utils.GetEnv("LB_ENDPOINT_LIST", "")
	endpoints := strings.Split(lbEnpointList, ",")

	http2httpsVal, err := strconv.Atoi(utils.GetEnv("HTTP2HTTPS", "0"))
	http2httpsVal = Coalesce(http2httpsVal, 0, err != nil).(int)
	http2https := http2httpsVal == 1

	redirectStatusCode, err := strconv.Atoi(utils.GetEnv("REDIRECT_STATUS_CODE", "301"))
	redirectStatusCode = Coalesce(redirectStatusCode, 301, redirectStatusCode == 0 || err != nil).(int)

	Config.Server.Forwarding.Host = Coalesce(Config.Server.Forwarding.Host, utils.GetEnv("FORWARD_HOST", ""), Config.Server.Forwarding.Host == "").(string)
	Config.Server.Forwarding.Scheme = Coalesce(Config.Server.Forwarding.Scheme, utils.GetEnv("FORWARD_SCHEME", ""), Config.Server.Forwarding.Scheme == "").(string)
	Config.Server.Forwarding.Endpoints = Coalesce(Config.Server.Forwarding.Endpoints, endpoints, len(Config.Server.Forwarding.Endpoints) == 0).([]string)
	Config.Server.Forwarding.HTTP2HTTPS = Coalesce(Config.Server.Forwarding.HTTP2HTTPS, http2https, !Config.Server.Forwarding.HTTP2HTTPS).(bool)
	Config.Server.Forwarding.RedirectStatusCode = Coalesce(Config.Server.Forwarding.RedirectStatusCode, redirectStatusCode, Config.Server.Forwarding.RedirectStatusCode == 0).(int)

	// --- Cache

	cacheDb, err := strconv.Atoi(utils.GetEnv("REDIS_DB", "0"))
	cacheDb = Coalesce(cacheDb, 0, err != nil).(int)

	ttlSecs, err := strconv.Atoi(utils.GetEnv("DEFAULT_TTL", "0"))
	ttlSecs = Coalesce(ttlSecs, 0, err != nil).(int)

	statuses := utils.GetEnv("CACHE_ALLOWED_STATUSES", "200,301,302")
	statusList := strings.Split(statuses, ",")

	methods := utils.GetEnv("CACHE_ALLOWED_METHODS", "HEAD,GET")
	methodList := strings.Split(methods, ",")

	Config.Cache.Host = Coalesce(Config.Cache.Host, utils.GetEnv("REDIS_HOST", ""), Config.Cache.Host == "").(string)
	Config.Cache.Port = Coalesce(Config.Cache.Port, utils.GetEnv("REDIS_PORT", "6379"), Config.Cache.Port == "").(string)
	Config.Cache.Password = Coalesce(Config.Cache.Password, utils.GetEnv("REDIS_PASSWORD", ""), Config.Cache.Password == "").(string)
	Config.Cache.DB = Coalesce(Config.Cache.DB, cacheDb, Config.Cache.DB == 0).(int)
	Config.Cache.TTL = Coalesce(Config.Cache.TTL, ttlSecs, Config.Cache.TTL == 0).(int)
	Config.Cache.AllowedStatuses = Coalesce(Config.Cache.AllowedStatuses, statusList, len(Config.Cache.AllowedStatuses) == 0).([]string)
	Config.Cache.AllowedMethods = Coalesce(Config.Cache.AllowedMethods, methodList, len(Config.Cache.AllowedMethods) == 0).([]string)
	Config.Cache.AllowedMethods = append(Config.Cache.AllowedMethods, "HEAD", "GET")
	Config.Cache.AllowedMethods = utils.Unique(Config.Cache.AllowedMethods)

	// TODO: split in 2 methods
	configAsYaml, err := yaml.Marshal(Config)
	if err == nil {
		log.Info("Config Settings:\n")
		log.Info(string(configAsYaml))
	}
}

// GetForwarding - Returns the forwarding configs.
func GetForwarding() Forward {
	return Config.Server.Forwarding
}

// GetPortHTTPS - Returns the HTTPS port
func GetPortHTTPS() string {
	return Config.Server.Port.HTTPS
}
