package config

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"crypto/tls"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/sony/gobreaker"
)

// Config - Holds the server configuration
var Config Configuration
var cb map[string]*gobreaker.CircuitBreaker
var allowedSchemes = map[string]string{"HTTP": "http", "HTTPS": "https"}

// Configuration - Defines the server configuration
type Configuration struct {
	Server         Server
	Cache          Cache
	CircuitBreaker CircuitBreaker
	Domains        Domains
	Log            Log
}

// Domains - Overrides per domain
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
	Override *tls.Config
}

// Forward - Defines the forwarding settings
type Forward struct {
	Host               string
	Port               string
	Scheme             string
	Endpoints          []string
	InsecureBridge     bool
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

type Log struct {
	TimeFormat string
	Format     string
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
				Override: &tls.Config{
					// TODO: handle this
					// Use modern tls mode https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility
					// NextProtos: []string{"h2", "http/1.1"},
					// Only use curves which have assembly implementations
					// https://github.com/golang/go/tree/master/src/crypto/elliptic
					CurvePreferences: []tls.CurveID{
						tls.CurveP256,
					},
					MinVersion: tls.VersionTLS12,
					MaxVersion: tls.VersionTLS13,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
						tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
						// needed by HTTP/2
						tls.TLS_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
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
				HTTP2HTTPS:         false,
				InsecureBridge:     false,
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
		Log: Log{
			TimeFormat: "2006/01/02 15:04:05",
			Format: `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status`,
		}
	}
}

func normalizeScheme(scheme string) string {
	schemeUpper := strings.ToUpper(scheme)
	if val, ok := allowedSchemes[schemeUpper]; ok {
		return val
	}

	return ""
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
				Read:       utils.ConvertToDuration(utils.GetEnv("TIMEOUT_READ", "")),
				ReadHeader: utils.ConvertToDuration(utils.GetEnv("TIMEOUT_READ_HEADER", "")),
				Write:      utils.ConvertToDuration(utils.GetEnv("TIMEOUT_WRITE", "")),
				Idle:       utils.ConvertToDuration(utils.GetEnv("TIMEOUT_IDLE", "")),
				Handler:    utils.ConvertToDuration(utils.GetEnv("TIMEOUT_HANDLER", "")),
			},
			Forwarding: Forward{
				Host:               utils.GetEnv("FORWARD_HOST", ""),
				Port:               utils.GetEnv("FORWARD_PORT", ""),
				Scheme:             normalizeScheme(utils.GetEnv("FORWARD_SCHEME", "")),
				Endpoints:          strings.Split(utils.GetEnv("LB_ENDPOINT_LIST", ""), ","),
				HTTP2HTTPS:         utils.GetEnv("HTTP2HTTPS", "") == "1",
				RedirectStatusCode: utils.ConvertToInt(utils.GetEnv("REDIRECT_STATUS_CODE", "")),
			},
			GZip: utils.GetEnv("GZIP_ENABLED", "") == "1",
		},
		Cache: Cache{
			Host:            utils.GetEnv("REDIS_HOST", ""),
			Port:            utils.GetEnv("REDIS_PORT", ""),
			Password:        utils.GetEnv("REDIS_PASSWORD", ""),
			DB:              utils.ConvertToInt(utils.GetEnv("REDIS_DB", "")),
			TTL:             utils.ConvertToInt(utils.GetEnv("DEFAULT_TTL", "")),
			AllowedStatuses: utils.ConvertToIntSlice(strings.Split(utils.GetEnv("CACHE_ALLOWED_STATUSES", ""), ",")),
			AllowedMethods:  strings.Split(utils.GetEnv("CACHE_ALLOWED_METHODS", ""), ","),
		},
	}
}

func getYamlConfig(file string) Configuration {
	YamlConfig := Configuration{}

	data, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		log.Warnf("Cannot read file %s: %s\n", file, err)
	}
	err = yaml.Unmarshal([]byte(data), &YamlConfig)
	if err != nil {
		log.Warnf("Cannot unmarshal yaml: %s\n", err)
	}

	YamlConfig.Server.Forwarding.Scheme = normalizeScheme(YamlConfig.Server.Forwarding.Scheme)

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

// CopyOverWith - Copies the Configuration over another (preserving not defined settings).
func CopyOverWith(base Configuration, overrides Configuration) Configuration {
	newConf := base

	// --- SERVER

	serverN := newConf.Server
	serverO := overrides.Server
	serverN.Port.HTTP = utils.Coalesce(serverO.Port.HTTP, serverN.Port.HTTP, serverO.Port.HTTP == "").(string)
	serverN.Port.HTTPS = utils.Coalesce(serverO.Port.HTTPS, serverN.Port.HTTPS, serverO.Port.HTTPS == "").(string)
	serverN.GZip = utils.Coalesce(serverO.GZip, serverN.GZip, !serverO.GZip).(bool)
	newConf.Server = serverN

	// --- TLS

	tlsN := newConf.Server.TLS
	tlsO := overrides.Server.TLS
	tlsN.Auto = utils.Coalesce(tlsO.Auto, tlsN.Auto, !tlsO.Auto).(bool)
	tlsN.Email = utils.Coalesce(tlsO.Email, tlsN.Email, tlsO.Email == "").(string)
	tlsN.CertFile = utils.Coalesce(tlsO.CertFile, tlsN.CertFile, tlsO.CertFile == "").(string)
	tlsN.KeyFile = utils.Coalesce(tlsO.KeyFile, tlsN.KeyFile, tlsO.KeyFile == "").(string)
	tlsN.Override = utils.Coalesce(tlsO.Override, tlsN.Override, tlsO.Override == nil).(*tls.Config)
	newConf.Server.TLS = tlsN

	// --- Timeout

	timeoutN := newConf.Server.Timeout
	timeoutO := overrides.Server.Timeout
	timeoutN.Read = utils.Coalesce(timeoutO.Read, timeoutN.Read, timeoutO.Read == 0).(time.Duration)
	timeoutN.ReadHeader = utils.Coalesce(timeoutO.ReadHeader, timeoutN.ReadHeader, timeoutO.ReadHeader == 0).(time.Duration)
	timeoutN.Write = utils.Coalesce(timeoutO.Write, timeoutN.Write, timeoutO.Write == 0).(time.Duration)
	timeoutN.Idle = utils.Coalesce(timeoutO.Idle, timeoutN.Idle, timeoutO.Idle == 0).(time.Duration)
	timeoutN.Handler = utils.Coalesce(timeoutO.Handler, timeoutN.Handler, timeoutO.Handler == 0).(time.Duration)
	newConf.Server.Timeout = timeoutN

	// --- Forwarding

	forwardingN := newConf.Server.Forwarding
	forwardingO := overrides.Server.Forwarding
	forwardingN.Host = utils.Coalesce(forwardingO.Host, forwardingN.Host, forwardingO.Host == "").(string)
	forwardingN.Port = utils.Coalesce(forwardingO.Port, forwardingN.Port, forwardingO.Port == "").(string)
	forwardingN.Scheme = utils.Coalesce(forwardingO.Scheme, forwardingN.Scheme, forwardingO.Scheme == "").(string)
	forwardingN.Endpoints = utils.Coalesce(forwardingO.Endpoints, forwardingN.Endpoints, len(forwardingO.Endpoints) == 0).([]string)
	forwardingN.HTTP2HTTPS = utils.Coalesce(forwardingO.HTTP2HTTPS, forwardingN.HTTP2HTTPS, !forwardingO.HTTP2HTTPS).(bool)
	forwardingN.InsecureBridge = utils.Coalesce(forwardingO.InsecureBridge, forwardingN.InsecureBridge, !forwardingO.InsecureBridge).(bool)
	forwardingN.RedirectStatusCode = utils.Coalesce(forwardingO.RedirectStatusCode, forwardingN.RedirectStatusCode, forwardingO.RedirectStatusCode == 0).(int)
	newConf.Server.Forwarding = forwardingN

	// --- Cache

	cacheN := newConf.Cache
	cacheO := overrides.Cache
	cacheN.Host = utils.Coalesce(cacheO.Host, newConf.Cache.Host, cacheO.Host == "").(string)
	cacheN.Port = utils.Coalesce(cacheO.Port, newConf.Cache.Port, cacheO.Port == "").(string)
	cacheN.Password = utils.Coalesce(cacheO.Password, newConf.Cache.Password, cacheO.Password == "").(string)
	cacheN.DB = utils.Coalesce(cacheO.DB, newConf.Cache.DB, cacheO.DB == 0).(int)
	cacheN.TTL = utils.Coalesce(cacheO.TTL, newConf.Cache.TTL, cacheO.TTL == 0).(int)
	cacheN.AllowedStatuses = utils.Coalesce(cacheO.AllowedStatuses, newConf.Cache.AllowedStatuses, len(cacheO.AllowedStatuses) == 0).([]int)
	cacheN.AllowedMethods = utils.Coalesce(cacheO.AllowedMethods, newConf.Cache.AllowedMethods, len(cacheO.AllowedMethods) == 0).([]string)

	cacheN.AllowedMethods = append(cacheN.AllowedMethods, "HEAD", "GET")
	cacheN.AllowedMethods = utils.Unique(cacheN.AllowedMethods)
	newConf.Cache = cacheN

	return newConf
}

// Print - Shows the current configuration.
func Print() {
	ObfuscatedConfig := Config
	ObfuscatedConfig.Cache.Password = ""
	for k, v := range ObfuscatedConfig.Domains {
		v.Cache.Password = ""
		ObfuscatedConfig.Domains[k] = v
	}
	log.Debug("Config Settings:\n")
	log.Debugf("%+v\n", ObfuscatedConfig)
}

// GetDomains - Returns a list of domains.
func GetDomains() []string {
	domains := make([]string, 0, len(Config.Domains))
	for _, v := range Config.Domains {
		domains = append(domains, v.Server.Forwarding.Host)
	}

	return domains
}

// DomainConf - Returns the configuration for the requested domain.
func DomainConf(domain string) *Configuration {
	domainParts := strings.Split(domain, ":")
	cleanedDomain := domainParts[0]

	for _, v := range Config.Domains {
		if v.Server.Forwarding.Host == cleanedDomain {
			return &v
		}
	}

	if Config.Server.Forwarding.Host == cleanedDomain {
		return &Config
	}

	return nil
}

// InitCircuitBreaker - Initialise the Circuit Breaker.
func InitCircuitBreaker(name string, config CircuitBreaker) {
	st := gobreaker.Settings{
		Name:        name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= config.Threshold && failureRatio >= config.FailureRate
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Warnf("Circuit Breaker - Changed from %s to %s", from.String(), to.String())
		},
	}

	if cb == nil {
		cb = make(map[string]*gobreaker.CircuitBreaker)
	}

	cb[name] = gobreaker.NewCircuitBreaker(st)
}

// CB - Returns instance of gobreaker.CircuitBreaker.
func CB(name string) *gobreaker.CircuitBreaker {
	if val, ok := cb[name]; ok {
		return val
	}

	log.Warnf("Missing circuit breaker for %s", name)
	return nil
}
