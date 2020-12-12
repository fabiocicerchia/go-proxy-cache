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
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
	"github.com/fabiocicerchia/go-proxy-cache/utils/convert"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

// Configuration - Defines the server configuration
type Configuration struct {
	Server         Server                        `yaml:"server"`
	Cache          Cache                         `yaml:"cache"`
	CircuitBreaker circuitbreaker.CircuitBreaker `yaml:"circuit_breaker"`
	Domains        Domains                       `yaml:"domains"`
	Log            Log                           `yaml:"log"`
}

// Domains - Overrides per domain
type Domains map[string]Configuration

// Server - Defines basic info for the server
type Server struct {
	Port        Port     `yaml:"port"`
	TLS         TLS      `yaml:"tls"`
	Timeout     Timeout  `yaml:"timeout"`
	Upstream    Upstream `yaml:"upstream"`
	GZip        bool     `yaml:"gzip"`
	Healthcheck bool     `yaml:"healthcheck"`
}

// Port - Defines the listening ports per protocol
type Port struct {
	HTTP  string `yaml:"http"`
	HTTPS string `yaml:"https"`
}

// TLS - Defines the configuration for SSL/TLS
type TLS struct {
	Auto     bool        `yaml:"auto"`
	Email    string      `yaml:"email"`
	CertFile string      `yaml:"cert_file"`
	KeyFile  string      `yaml:"key_file"`
	Override *tls.Config `yaml:"override"`
}

// Upstream - Defines the upstream settings
type Upstream struct {
	Host               string   `yaml:"host"`
	Port               string   `yaml:"port"`
	Scheme             string   `yaml:"scheme"`
	Endpoints          []string `yaml:"endpoints"`
	InsecureBridge     bool     `yaml:"insecure_bridge"`
	HTTP2HTTPS         bool     `yaml:"http_to_https"`
	RedirectStatusCode int      `yaml:"redirect_status_code"`
}

// Timeout - Defines the server timeouts
type Timeout struct {
	Read       time.Duration `yaml:"read"`
	ReadHeader time.Duration `yaml:"read_header"`
	Write      time.Duration `yaml:"write"`
	Idle       time.Duration `yaml:"idle"`
	Handler    time.Duration `yaml:"handler"`
}

// Cache - Defines the config for the cache backend
type Cache struct {
	Host            string   `yaml:"host"`
	Port            string   `yaml:"port"`
	Password        string   `yaml:"password"`
	DB              int      `yaml:"db"`
	TTL             int      `yaml:"ttl"`
	AllowedStatuses []int    `yaml:"allowed_statuses"`
	AllowedMethods  []string `yaml:"allowed_methods"`
}

// Log - Defines the config for the logs
type Log struct {
	TimeFormat string `yaml:"time_format"`
	Format     string `yaml:"format"`
}

// Config - Holds the server configuration
var Config Configuration = Configuration{
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
		Upstream: Upstream{
			HTTP2HTTPS:         false,
			InsecureBridge:     false,
			RedirectStatusCode: 301,
		},
		GZip:        false,
		Healthcheck: true,
	},
	Cache: Cache{
		Port:            "6379",
		DB:              0,
		TTL:             0,
		AllowedStatuses: []int{200, 301, 302},
		AllowedMethods:  []string{"HEAD", "GET"},
	},
	CircuitBreaker: circuitbreaker.CircuitBreaker{
		Threshold:   2,                // after 2nd request, if meet FailureRate goes open.
		FailureRate: 0.5,              // 1 out of 2 fails, or more
		Interval:    0,                // doesn't clears counts
		Timeout:     60 * time.Second, // clears state after 60s
		MaxRequests: 1,
	},
	Log: Log{
		TimeFormat: "2006/01/02 15:04:05",
		Format:     `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status`,
	},
}

var allowedSchemes = map[string]string{"HTTP": "http", "HTTPS": "https"}

func normalizeScheme(scheme string) string {
	schemeUpper := strings.ToUpper(scheme)
	if val, ok := allowedSchemes[schemeUpper]; ok {
		return val
	}

	return ""
}

func getEnvConfig() Configuration {
	EnvConfig := Configuration{}

	EnvConfig.Server.Port.HTTP = utils.GetEnv("SERVER_HTTP_PORT", "")
	EnvConfig.Server.Port.HTTPS = utils.GetEnv("SERVER_HTTPS_PORT", "")

	EnvConfig.Server.TLS.Auto = utils.GetEnv("TLS_AUTO_CERT", "") == "1"
	EnvConfig.Server.TLS.Email = utils.GetEnv("TLS_EMAIL", "")
	EnvConfig.Server.TLS.CertFile = utils.GetEnv("TLS_CERT_FILE", "")
	EnvConfig.Server.TLS.KeyFile = utils.GetEnv("TLS_KEY_FILE", "")

	EnvConfig.Server.Timeout.Read = convert.ToDuration(utils.GetEnv("TIMEOUT_READ", ""))
	EnvConfig.Server.Timeout.ReadHeader = convert.ToDuration(utils.GetEnv("TIMEOUT_READ_HEADER", ""))
	EnvConfig.Server.Timeout.Write = convert.ToDuration(utils.GetEnv("TIMEOUT_WRITE", ""))
	EnvConfig.Server.Timeout.Idle = convert.ToDuration(utils.GetEnv("TIMEOUT_IDLE", ""))
	EnvConfig.Server.Timeout.Handler = convert.ToDuration(utils.GetEnv("TIMEOUT_HANDLER", ""))

	EnvConfig.Server.Upstream.Host = utils.GetEnv("FORWARD_HOST", "")
	EnvConfig.Server.Upstream.Port = utils.GetEnv("FORWARD_PORT", "")
	EnvConfig.Server.Upstream.Scheme = normalizeScheme(utils.GetEnv("FORWARD_SCHEME", ""))
	EnvConfig.Server.Upstream.Endpoints = strings.Split(utils.GetEnv("LB_ENDPOINT_LIST", ""), ",")
	EnvConfig.Server.Upstream.HTTP2HTTPS = utils.GetEnv("HTTP2HTTPS", "") == "1"
	EnvConfig.Server.Upstream.RedirectStatusCode = convert.ToInt(utils.GetEnv("REDIRECT_STATUS_CODE", ""))

	EnvConfig.Server.GZip = utils.GetEnv("GZIP_ENABLED", "") == "1"

	EnvConfig.Cache.Host = utils.GetEnv("REDIS_HOST", "")
	EnvConfig.Cache.Port = utils.GetEnv("REDIS_PORT", "")
	EnvConfig.Cache.Password = utils.GetEnv("REDIS_PASSWORD", "")
	EnvConfig.Cache.DB = convert.ToInt(utils.GetEnv("REDIS_DB", ""))
	EnvConfig.Cache.TTL = convert.ToInt(utils.GetEnv("DEFAULT_TTL", ""))
	EnvConfig.Cache.AllowedStatuses = convert.ToIntSlice(strings.Split(utils.GetEnv("CACHE_ALLOWED_STATUSES", ""), ","))
	EnvConfig.Cache.AllowedMethods = strings.Split(utils.GetEnv("CACHE_ALLOWED_METHODS", ""), ",")

	return EnvConfig
}

func getYamlConfig(file string) (Configuration, error) {
	YamlConfig := Configuration{}

	data, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return YamlConfig, err
	}

	err = yaml.UnmarshalStrict([]byte(data), &YamlConfig)

	if err != nil {
		return YamlConfig, err
	}

	YamlConfig.Server.Upstream.Scheme = normalizeScheme(YamlConfig.Server.Upstream.Scheme)

	return YamlConfig, err
}

// InitConfigFromFileOrEnv - Init the configuration in sequence: from a YAML file, from environment variables,
// then defaults.
func InitConfigFromFileOrEnv(file string) {
	Config = CopyOverWith(Config, getEnvConfig())

	var YamlConfig Configuration
	_, err := os.Stat(file)
	if !os.IsNotExist(err) {
		YamlConfig, err = getYamlConfig(file)
		if err != nil {
			log.Fatalf("Cannot unmarshal YAML: %s\n", err)
		}
		Config = CopyOverWith(Config, YamlConfig)
	}

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

// Validate - Validate a YAML config file is syntactically valid.
func Validate(file string) (bool, error) {
	_, err := getYamlConfig(file)
	return err != nil, err
}

// CopyOverWith - Copies the Configuration over another (preserving not defined settings).
func CopyOverWith(base Configuration, overrides Configuration) Configuration {
	newConf := base

	// --- SERVER
	newConf.Server.Port.HTTP = utils.Coalesce(overrides.Server.Port.HTTP, newConf.Server.Port.HTTP, overrides.Server.Port.HTTP == "").(string)
	newConf.Server.Port.HTTPS = utils.Coalesce(overrides.Server.Port.HTTPS, newConf.Server.Port.HTTPS, overrides.Server.Port.HTTPS == "").(string)
	newConf.Server.GZip = utils.Coalesce(overrides.Server.GZip, newConf.Server.GZip, !overrides.Server.GZip).(bool)

	// --- TLS
	newConf.Server.TLS.Auto = utils.Coalesce(overrides.Server.TLS.Auto, newConf.Server.TLS.Auto, !overrides.Server.TLS.Auto).(bool)
	newConf.Server.TLS.Email = utils.Coalesce(overrides.Server.TLS.Email, newConf.Server.TLS.Email, overrides.Server.TLS.Email == "").(string)
	newConf.Server.TLS.CertFile = utils.Coalesce(overrides.Server.TLS.CertFile, newConf.Server.TLS.CertFile, overrides.Server.TLS.CertFile == "").(string)
	newConf.Server.TLS.KeyFile = utils.Coalesce(overrides.Server.TLS.KeyFile, newConf.Server.TLS.KeyFile, overrides.Server.TLS.KeyFile == "").(string)
	newConf.Server.TLS.Override = utils.Coalesce(overrides.Server.TLS.Override, newConf.Server.TLS.Override, overrides.Server.TLS.Override == nil).(*tls.Config)

	// --- Timeout
	newConf.Server.Timeout.Read = utils.Coalesce(overrides.Server.Timeout.Read, newConf.Server.Timeout.Read, overrides.Server.Timeout.Read == 0).(time.Duration)
	newConf.Server.Timeout.ReadHeader = utils.Coalesce(overrides.Server.Timeout.ReadHeader, newConf.Server.Timeout.ReadHeader, overrides.Server.Timeout.ReadHeader == 0).(time.Duration)
	newConf.Server.Timeout.Write = utils.Coalesce(overrides.Server.Timeout.Write, newConf.Server.Timeout.Write, overrides.Server.Timeout.Write == 0).(time.Duration)
	newConf.Server.Timeout.Idle = utils.Coalesce(overrides.Server.Timeout.Idle, newConf.Server.Timeout.Idle, overrides.Server.Timeout.Idle == 0).(time.Duration)
	newConf.Server.Timeout.Handler = utils.Coalesce(overrides.Server.Timeout.Handler, newConf.Server.Timeout.Handler, overrides.Server.Timeout.Handler == 0).(time.Duration)

	// --- Upstream
	newConf.Server.Upstream.Host = utils.Coalesce(overrides.Server.Upstream.Host, newConf.Server.Upstream.Host, overrides.Server.Upstream.Host == "").(string)
	newConf.Server.Upstream.Port = utils.Coalesce(overrides.Server.Upstream.Port, newConf.Server.Upstream.Port, overrides.Server.Upstream.Port == "").(string)
	newConf.Server.Upstream.Scheme = utils.Coalesce(overrides.Server.Upstream.Scheme, newConf.Server.Upstream.Scheme, overrides.Server.Upstream.Scheme == "").(string)
	newConf.Server.Upstream.Endpoints = utils.Coalesce(overrides.Server.Upstream.Endpoints, newConf.Server.Upstream.Endpoints, len(overrides.Server.Upstream.Endpoints) == 0).([]string)
	newConf.Server.Upstream.HTTP2HTTPS = utils.Coalesce(overrides.Server.Upstream.HTTP2HTTPS, newConf.Server.Upstream.HTTP2HTTPS, !overrides.Server.Upstream.HTTP2HTTPS).(bool)
	newConf.Server.Upstream.InsecureBridge = utils.Coalesce(overrides.Server.Upstream.InsecureBridge, newConf.Server.Upstream.InsecureBridge, !overrides.Server.Upstream.InsecureBridge).(bool)
	newConf.Server.Upstream.RedirectStatusCode = utils.Coalesce(overrides.Server.Upstream.RedirectStatusCode, newConf.Server.Upstream.RedirectStatusCode, overrides.Server.Upstream.RedirectStatusCode == 0).(int)

	// --- Cache
	newConf.Cache.Host = utils.Coalesce(overrides.Cache.Host, newConf.Cache.Host, overrides.Cache.Host == "").(string)
	newConf.Cache.Port = utils.Coalesce(overrides.Cache.Port, newConf.Cache.Port, overrides.Cache.Port == "").(string)
	newConf.Cache.Password = utils.Coalesce(overrides.Cache.Password, newConf.Cache.Password, overrides.Cache.Password == "").(string)
	newConf.Cache.DB = utils.Coalesce(overrides.Cache.DB, newConf.Cache.DB, overrides.Cache.DB == 0).(int)
	newConf.Cache.TTL = utils.Coalesce(overrides.Cache.TTL, newConf.Cache.TTL, overrides.Cache.TTL == 0).(int)
	newConf.Cache.AllowedStatuses = utils.Coalesce(overrides.Cache.AllowedStatuses, newConf.Cache.AllowedStatuses, len(overrides.Cache.AllowedStatuses) == 0).([]int)
	newConf.Cache.AllowedMethods = utils.Coalesce(overrides.Cache.AllowedMethods, newConf.Cache.AllowedMethods, len(overrides.Cache.AllowedMethods) == 0).([]string)

	newConf.Cache.AllowedMethods = append(newConf.Cache.AllowedMethods, "HEAD", "GET")
	newConf.Cache.AllowedMethods = slice.Unique(newConf.Cache.AllowedMethods)

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
func GetDomains() []map[string]string {
	// TODO: What if there's no domains only main config?!
	domains := make([]map[string]string, 0, len(Config.Domains))
	for _, v := range Config.Domains {
		d := map[string]string{
			"host":   v.Server.Upstream.Host,
			"scheme": v.Server.Upstream.Scheme,
		}
		domains = append(domains, d)
	}

	return domains
}

// DomainConf - Returns the configuration for the requested domain.
func DomainConf(domain string, scheme string) *Configuration {
	domainParts := strings.Split(domain, ":")
	cleanedDomain := domainParts[0]

	// TODO: Use memoization

	// First round: host & scheme
	for _, v := range Config.Domains {
		if v.Server.Upstream.Host == cleanedDomain && v.Server.Upstream.Scheme == scheme {
			return &v
		}
	}

	// Second round: host
	for _, v := range Config.Domains {
		if v.Server.Upstream.Host == cleanedDomain {
			return &v
		}
	}

	// Third round: global
	if Config.Server.Upstream.Host == cleanedDomain {
		return &Config
	}

	return nil
}
