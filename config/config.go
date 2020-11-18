package config

import (
	"io/ioutil"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
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
	TTL        int
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
	Host      string
	Scheme    string
	Endpoints []string
}

// Timeout - Defines the server timeouts
type Timeout struct {
	Read       int
	Write      int
	Idle       int
	ReadHeader int
	Handler    int
}

// Cache - Defines the config for the cache backend
type Cache struct {
	Host            string
	Port            string
	Password        string
	DB              int
	AllowedStatuses []string
	AllowedMethods  []string
}

// InitConfigFromFileOrEnv - Init the configuration in sequence: from a YAML file, from environment variables, then defaults.
func InitConfigFromFileOrEnv(file string) {
	Config = Configuration{}

	data, _ := ioutil.ReadFile(file)
	_ = yaml.Unmarshal([]byte(data), &Config)

	// --- Server

	ttlSecs, _ := strconv.Atoi(utils.GetEnv("DEFAULT_TTL", "0"))

	if Config.Server.Port.HTTP == "" {
		Config.Server.Port.HTTP = utils.GetEnv("SERVER_HTTP_PORT", "80")
	}
	if Config.Server.Port.HTTPS == "" {
		Config.Server.Port.HTTPS = utils.GetEnv("SERVER_HTTPS_PORT", "443")
	}
	if Config.Server.TTL == 0 {
		Config.Server.TTL = ttlSecs
	}

	// --- TLS

	autoTLSCertVal, _ := strconv.Atoi(utils.GetEnv("TLS_AUTO_CERT", "0"))
	autoTLSCert := autoTLSCertVal == 1

	if !Config.Server.TLS.Auto {
		Config.Server.TLS.Auto = autoTLSCert
	}
	if Config.Server.TLS.Email == "" {
		Config.Server.TLS.Email = utils.GetEnv("TLS_EMAIL", "")
	}
	if Config.Server.TLS.CertFile == "" {
		Config.Server.TLS.CertFile = utils.GetEnv("TLS_CERT_FILE", "")
	}
	if Config.Server.TLS.KeyFile == "" {
		Config.Server.TLS.KeyFile = utils.GetEnv("TLS_KEY_FILE", "")
	}

	// --- Timeout

	timeoutRead, err := strconv.Atoi(utils.GetEnv("TIMEOUT_READ", ""))
	if timeoutRead == 0 || err != nil {
		timeoutRead = 5
	}
	timeoutWrite, err := strconv.Atoi(utils.GetEnv("TIMEOUT_WRITE", ""))
	if timeoutWrite == 0 || err != nil {
		timeoutWrite = 5
	}
	timeoutIdle, err := strconv.Atoi(utils.GetEnv("TIMEOUT_IDLE", ""))
	if timeoutIdle == 0 || err != nil {
		timeoutIdle = 30
	}
	timeoutReadHeader, err := strconv.Atoi(utils.GetEnv("TIMEOUT_READ_HEADER", ""))
	if timeoutReadHeader == 0 || err != nil {
		timeoutReadHeader = 2
	}
	timeoutHandler, err := strconv.Atoi(utils.GetEnv("TIMEOUT_HANDLER", ""))
	if timeoutHandler == 0 || err != nil {
		timeoutHandler = 5
	}

	if Config.Server.Timeout.Read == 0 {
		Config.Server.Timeout.Read = timeoutRead
	}
	if Config.Server.Timeout.Write == 0 {
		Config.Server.Timeout.Write = timeoutWrite
	}
	if Config.Server.Timeout.Idle == 0 {
		Config.Server.Timeout.Idle = timeoutIdle
	}
	if Config.Server.Timeout.ReadHeader == 0 {
		Config.Server.Timeout.ReadHeader = timeoutReadHeader
	}
	if Config.Server.Timeout.Handler == 0 {
		Config.Server.Timeout.Handler = timeoutHandler
	}

	// --- Forwarding

	lbEnpointList := utils.GetEnv("LB_ENDPOINT_LIST", "")
	endpoints := strings.Split(lbEnpointList, ",")

	if Config.Server.Forwarding.Host == "" {
		Config.Server.Forwarding.Host = utils.GetEnv("FORWARD_HOST", "")
	}
	if Config.Server.Forwarding.Scheme == "" {
		Config.Server.Forwarding.Scheme = utils.GetEnv("FORWARD_SCHEME", "")
	}
	if len(Config.Server.Forwarding.Endpoints) == 0 {
		Config.Server.Forwarding.Endpoints = endpoints
	}

	// --- Cache

	cacheDb, err := strconv.Atoi(utils.GetEnv("REDIS_DB", "0"))
	if err != nil {
		cacheDb = 0
	}

	statuses := utils.GetEnv("CACHE_ALLOWED_STATUSES", "200,301,302")
	statusList := strings.Split(statuses, ",")

	methods := utils.GetEnv("CACHE_ALLOWED_METHODS", "HEAD,GET")
	methodList := strings.Split(methods, ",")

	if Config.Cache.Host == "" {
		Config.Cache.Host = utils.GetEnv("REDIS_HOST", "")
	}
	if Config.Cache.Port == "" {
		Config.Cache.Port = utils.GetEnv("REDIS_PORT", "6379")
	}
	if Config.Cache.Password == "" {
		Config.Cache.Password = utils.GetEnv("REDIS_PASSWORD", "")
	}
	if Config.Cache.DB == 0 {
		Config.Cache.DB = cacheDb
	}
	if len(Config.Cache.AllowedStatuses) == 0 {
		Config.Cache.AllowedStatuses = statusList
	}
	if len(Config.Cache.AllowedMethods) == 0 {
		Config.Cache.AllowedMethods = methodList
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
