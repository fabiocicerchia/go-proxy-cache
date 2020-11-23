//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache
package config

import (
	"crypto/tls"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	log "github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

// Config - Holds the server configuration
var Config Configuration
var cb *gobreaker.CircuitBreaker

// Configuration - Defines the server configuration
type Configuration struct {
	Server         Server
	Cache          Cache
	CircuitBreaker CircuitBreaker
	Domains        Domains
}

type Domains map[string]Configuration

// Server - Defines basic info for the server
type Server struct {
	Port       Port
	TLS        TLS
	Timeout    Timeout
	Forwarding Forward
	GZip       bool
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
	Override tls.Config
}

// Forward - Defines the forwarding settings
type Forward struct {
	Host               string
	Port               string
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
	AllowedStatuses []int
	AllowedMethods  []string
}

// CircuitBreaker - Settings for redis circuit breaker.
type CircuitBreaker struct {
	Threshold   uint32
	FailureRate float64
	Interval    time.Duration
	Timeout     time.Duration
	MaxRequests uint32
}

func getDefaultConfig() Configuration {
	return Configuration{
		Server: Server{
			Port: Port{
				HTTP:  "80",
				HTTPS: "443",
			},
			TLS: TLS{
				Auto:     false,
				Email:    "",
				CertFile: "",
				KeyFile:  "",
				Override: tls.Config{
					// Causes servers to use Go's default ciphersuite preferences,
					// which are tuned to avoid attacks. Does nothing on clients.
					PreferServerCipherSuites: true,
					// Only use curves which have assembly implementations
					CurvePreferences: []tls.CurveID{
						tls.CurveP256,
						tls.X25519, // Go 1.8 only
					},
					MinVersion: tls.VersionTLS12,
					MaxVersion: tls.VersionTLS13,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
						tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						// Best disabled, as they don't provide Forward Secrecy,
						// but might be necessary for some clients
						// tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						// tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
					},
				},
			},
			Timeout: Timeout{
				Read:       5 * time.Second,
				ReadHeader: 2 * time.Second,
				Write:      5 * time.Second,
				Idle:       20 * time.Second,
				Handler:    5 * time.Second,
			},
			Forwarding: Forward{
				HTTP2HTTPS:         true,
				RedirectStatusCode: 301,
			},
			GZip: false,
		},
		Cache: Cache{
			Port:            "6379",
			DB:              0,
			TTL:             0,
			AllowedStatuses: []int{200, 301, 302},
			AllowedMethods:  []string{"HEAD", "GET"},
		},
		CircuitBreaker: CircuitBreaker{
			Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
			FailureRate: 0.5,              // 1 out of 2 fails, or more
			Interval:    0,                // doesn't clears counts
			Timeout:     60 * time.Second, // clears state after 60s
			MaxRequests: 1,
		},
	}
}

func getEnvConfig() Configuration {
	return Configuration{
		Server: Server{
			Port: Port{
				HTTP:  utils.GetEnv("SERVER_HTTP_PORT", ""),
				HTTPS: utils.GetEnv("SERVER_HTTPS_PORT", ""),
			},
			TLS: TLS{
				Auto:     utils.GetEnv("TLS_AUTO_CERT", "") == "1",
				Email:    utils.GetEnv("TLS_EMAIL", ""),
				CertFile: utils.GetEnv("TLS_CERT_FILE", ""),
				KeyFile:  utils.GetEnv("TLS_KEY_FILE", ""),
			},
			Timeout: Timeout{
				Read:       ConvertToDuration(utils.GetEnv("TIMEOUT_READ", "")),
				ReadHeader: ConvertToDuration(utils.GetEnv("TIMEOUT_READ_HEADER", "")),
				Write:      ConvertToDuration(utils.GetEnv("TIMEOUT_WRITE", "")),
				Idle:       ConvertToDuration(utils.GetEnv("TIMEOUT_IDLE", "")),
				Handler:    ConvertToDuration(utils.GetEnv("TIMEOUT_HANDLER", "")),
			},
			Forwarding: Forward{
				Host:               utils.GetEnv("FORWARD_HOST", ""),
				Port:               utils.GetEnv("FORWARD_PORT", ""),
				Scheme:             utils.GetEnv("FORWARD_SCHEME", ""),
				Endpoints:          strings.Split(utils.GetEnv("LB_ENDPOINT_LIST", ""), ","),
				HTTP2HTTPS:         utils.GetEnv("HTTP2HTTPS", "") == "1",
				RedirectStatusCode: ConvertToInt(utils.GetEnv("REDIRECT_STATUS_CODE", "")),
			},
			GZip: utils.GetEnv("GZIP_ENABLED", "") == "1",
		},
		Cache: Cache{
			Host:            utils.GetEnv("REDIS_HOST", ""),
			Port:            utils.GetEnv("REDIS_PORT", ""),
			Password:        utils.GetEnv("REDIS_PASSWORD", ""),
			DB:              ConvertToInt(utils.GetEnv("REDIS_DB", "")),
			TTL:             ConvertToInt(utils.GetEnv("DEFAULT_TTL", "")),
			AllowedStatuses: ConvertToIntSlice(strings.Split(utils.GetEnv("CACHE_ALLOWED_STATUSES", ""), ",")),
			AllowedMethods:  strings.Split(utils.GetEnv("CACHE_ALLOWED_METHODS", ""), ","),
		},
	}
}

func getYamlConfig(file string) Configuration {
	YamlConfig := Configuration{}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Warnf("Cannot read file %s: %s\n", file, err)
	}
	err = yaml.Unmarshal([]byte(data), &YamlConfig)
	if err != nil {
		log.Warnf("Cannot unmarshal yaml: %s\n", err)
	}

	return YamlConfig
}

// InitConfigFromFileOrEnv - Init the configuration in sequence: from a YAML file, from environment variables,
// then defaults.
func InitConfigFromFileOrEnv(file string) {
	Config = Configuration{}
	Config = CopyOverWith(Config, getDefaultConfig())
	Config = CopyOverWith(Config, getEnvConfig())
	YamlConfig := getYamlConfig(file)
	Config = CopyOverWith(Config, YamlConfig)

	// allow only the config file to specify overrides per domain
	Config.Domains = YamlConfig.Domains

	// DOMAINS

	if Config.Domains != nil {
		domains := Config.Domains
		for k, v := range domains {
			baseConf := Config
			domain := CopyOverWith(baseConf, v)
			domain.Domains = Domains{}
			domains[k] = domain
		}
		Config.Domains = domains
	}
}

func CopyOverWith(base Configuration, overrides Configuration) Configuration {
	newConf := base

	// --- SERVER

	serverN := newConf.Server
	serverO := overrides.Server
	serverN.Port.HTTP = Coalesce(serverO.Port.HTTP, serverN.Port.HTTP, serverO.Port.HTTP == "").(string)
	serverN.Port.HTTPS = Coalesce(serverO.Port.HTTPS, serverN.Port.HTTPS, serverO.Port.HTTPS == "").(string)
	serverN.GZip = Coalesce(serverO.GZip, serverN.GZip, !serverO.GZip).(bool)
	newConf.Server = serverN

	// --- TLS

	tlsN := newConf.Server.TLS
	tlsO := overrides.Server.TLS
	tlsN.Auto = Coalesce(tlsO.Auto, tlsN.Auto, !tlsO.Auto).(bool)
	tlsN.Email = Coalesce(tlsO.Email, tlsN.Email, tlsO.Email == "").(string)
	tlsN.CertFile = Coalesce(tlsO.CertFile, tlsN.CertFile, tlsO.CertFile == "").(string)
	tlsN.KeyFile = Coalesce(tlsO.KeyFile, tlsN.KeyFile, tlsO.KeyFile == "").(string)
	newConf.Server.TLS = tlsN

	// --- Timeout

	timeoutN := newConf.Server.Timeout
	timeoutO := overrides.Server.Timeout
	timeoutN.Read = Coalesce(timeoutO.Read, timeoutN.Read, timeoutO.Read == 0).(time.Duration)
	timeoutN.ReadHeader = Coalesce(timeoutO.ReadHeader, timeoutN.ReadHeader, timeoutO.ReadHeader == 0).(time.Duration)
	timeoutN.Write = Coalesce(timeoutO.Write, timeoutN.Write, timeoutO.Write == 0).(time.Duration)
	timeoutN.Idle = Coalesce(timeoutO.Idle, timeoutN.Idle, timeoutO.Idle == 0).(time.Duration)
	timeoutN.Handler = Coalesce(timeoutO.Handler, timeoutN.Handler, timeoutO.Handler == 0).(time.Duration)
	newConf.Server.Timeout = timeoutN

	// --- Forwarding

	forwardingN := newConf.Server.Forwarding
	forwardingO := overrides.Server.Forwarding
	forwardingN.Host = Coalesce(forwardingO.Host, forwardingN.Host, forwardingO.Host == "").(string)
	forwardingN.Scheme = Coalesce(forwardingO.Scheme, forwardingN.Scheme, forwardingO.Scheme == "").(string)
	forwardingN.Endpoints = Coalesce(forwardingO.Endpoints, forwardingN.Endpoints, len(forwardingO.Endpoints) == 0).([]string)
	forwardingN.HTTP2HTTPS = Coalesce(forwardingO.HTTP2HTTPS, forwardingN.HTTP2HTTPS, !forwardingO.HTTP2HTTPS).(bool)
	forwardingN.RedirectStatusCode = Coalesce(forwardingO.RedirectStatusCode, forwardingN.RedirectStatusCode, forwardingO.RedirectStatusCode == 0).(int)
	newConf.Server.Forwarding = forwardingN

	// --- Cache

	cacheN := newConf.Cache
	cacheO := overrides.Cache
	cacheN.Host = Coalesce(cacheO.Host, newConf.Cache.Host, cacheO.Host == "").(string)
	cacheN.Port = Coalesce(cacheO.Port, newConf.Cache.Port, cacheO.Port == "").(string)
	cacheN.Password = Coalesce(cacheO.Password, newConf.Cache.Password, cacheO.Password == "").(string)
	cacheN.DB = Coalesce(cacheO.DB, newConf.Cache.DB, cacheO.DB == 0).(int)
	cacheN.TTL = Coalesce(cacheO.TTL, newConf.Cache.TTL, cacheO.TTL == 0).(int)
	cacheN.AllowedStatuses = Coalesce(cacheO.AllowedStatuses, newConf.Cache.AllowedStatuses, len(cacheO.AllowedStatuses) == 0).([]int)
	cacheN.AllowedMethods = Coalesce(cacheO.AllowedMethods, newConf.Cache.AllowedMethods, len(cacheO.AllowedMethods) == 0).([]string)

	cacheN.AllowedMethods = append(cacheN.AllowedMethods, "HEAD", "GET")
	cacheN.AllowedMethods = utils.Unique(cacheN.AllowedMethods)
	newConf.Cache = cacheN

	return newConf
}

// Print - Shows the current configuration.
func Print() {
	log.Info("Config Settings:\n")
	log.Infof("%+v\n", Config)
}

func GetDomains() []string {
	domains := make([]string, 0, len(Config.Domains))
	for _, v := range Config.Domains {
		domains = append(domains, v.Server.Forwarding.Host)
	}

	return domains
}

// DomainConf - Returns the configuration for the requested domain.
func DomainConf(domain string) *Configuration {
	for _, v := range Config.Domains {
		if v.Server.Forwarding.Host == domain {
			return &v
		}
	}

	if Config.Server.Forwarding.Host == domain {
		return &Config
	}

	return nil
}

// InitCircuitBreaker - Initialise the Circuit Breaker.
func InitCircuitBreaker(config CircuitBreaker) {
	var st gobreaker.Settings
	st.ReadyToTrip = func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.Requests >= config.Threshold && failureRatio >= config.FailureRate
	}
	st.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {
		log.Warnf("Circuit Breaker - Changed from %s to %s", from.String(), to.String())
	}
	st.Interval = config.Interval
	st.Timeout = config.Timeout
	st.MaxRequests = config.MaxRequests

	cb = gobreaker.NewCircuitBreaker(st)
}

// CB - Returns instance of gobreaker.CircuitBreaker.
func CB() *gobreaker.CircuitBreaker {
	return cb
}

// TODO" MOVE TO UTILS

// Coalesce - Returns the original value if the conditions is not met, fallback value otherwise.
func Coalesce(value interface{}, fallback interface{}, condition bool) interface{} {
	if condition {
		value = fallback
	}

	return value
}

func ConvertToDuration(value string) time.Duration {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return time.Duration(0)
	}
	return duration
}

func ConvertToInt(value string) int {
	val, _ := strconv.Atoi(value)
	return val
}

func ConvertToIntSlice(value []string) []int {
	var values []int
	for _, v := range value {
		values = append(values, ConvertToInt(v))
	}
	return values
}
