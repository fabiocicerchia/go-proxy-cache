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
	"fmt"
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

// DomainSet - Holds the uniqueness details of the domain
type DomainSet struct {
	Host   string
	Scheme string
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
var domainsCache map[string]*Configuration

func normalizeScheme(scheme string) string {
	schemeUpper := strings.ToUpper(scheme)
	if val, ok := allowedSchemes[schemeUpper]; ok {
		return val
	}

	return ""
}

func getEnvConfig() Configuration {
	EnvConfig := Configuration{}

	EnvConfig.Server.Port.HTTPS = utils.GetEnv("SERVER_HTTPS_PORT", "")
	EnvConfig.Server.Port.HTTP = utils.GetEnv("SERVER_HTTP_PORT", "")

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

	EnvConfig.Cache.DB = convert.ToInt(utils.GetEnv("REDIS_DB", ""))
	EnvConfig.Cache.Host = utils.GetEnv("REDIS_HOST", "")
	EnvConfig.Cache.Port = utils.GetEnv("REDIS_PORT", "")
	EnvConfig.Cache.Password = utils.GetEnv("REDIS_PASSWORD", "")
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
	Config.CopyOverWith(getEnvConfig(), nil)

	var YamlConfig Configuration
	_, err := os.Stat(file)
	if !os.IsNotExist(err) {
		YamlConfig, err = getYamlConfig(file)
		if err != nil {
			log.Fatalf("Cannot unmarshal YAML: %s\n", err)
			return
		}
		Config.CopyOverWith(YamlConfig, &file)
	}

	// allow only the config file to specify overrides per domain
	Config.Domains = YamlConfig.Domains

	// DOMAINS
	if Config.Domains != nil {
		domains := Config.Domains
		for k, v := range domains {
			domain := Config
			domain.CopyOverWith(v, &file)
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

func patchAbsFilePath(filePath string, relativeTo *string) string {
	abs, err := os.Getwd()

	if err == nil && relativeTo != nil && *relativeTo != "" {
		abs, err = filepath.Abs(*relativeTo)
		abs = filepath.Dir(abs)
	}

	if err == nil {
		if filePath != "" && !strings.HasPrefix(filePath, "/") {
			return filepath.Join(abs, filepath.Clean(filePath))
		}
	}

	return filePath
}

// CopyOverWith - Copies the Configuration over another (preserving not defined settings).
func (c *Configuration) CopyOverWith(overrides Configuration, file *string) {
	c.copyOverWithServer(overrides, file)
	c.copyOverWithTLS(overrides, file)
	c.copyOverWithTimeout(overrides, file)
	c.copyOverWithUpstream(overrides, file)
	c.copyOverWithCache(overrides, file)
}

// --- SERVER
func (c *Configuration) copyOverWithServer(overrides Configuration, file *string) {
	c.Server.Port.HTTP = utils.Coalesce(overrides.Server.Port.HTTP, c.Server.Port.HTTP, overrides.Server.Port.HTTP == "").(string)
	c.Server.Port.HTTPS = utils.Coalesce(overrides.Server.Port.HTTPS, c.Server.Port.HTTPS, overrides.Server.Port.HTTPS == "").(string)
	c.Server.GZip = utils.Coalesce(overrides.Server.GZip, c.Server.GZip, !overrides.Server.GZip).(bool)
}

// --- TLS
func (c *Configuration) copyOverWithTLS(overrides Configuration, file *string) {
	c.Server.TLS.Auto = utils.Coalesce(overrides.Server.TLS.Auto, c.Server.TLS.Auto, !overrides.Server.TLS.Auto).(bool)
	c.Server.TLS.Email = utils.Coalesce(overrides.Server.TLS.Email, c.Server.TLS.Email, overrides.Server.TLS.Email == "").(string)
	c.Server.TLS.CertFile = utils.Coalesce(overrides.Server.TLS.CertFile, c.Server.TLS.CertFile, overrides.Server.TLS.CertFile == "").(string)
	c.Server.TLS.KeyFile = utils.Coalesce(overrides.Server.TLS.KeyFile, c.Server.TLS.KeyFile, overrides.Server.TLS.KeyFile == "").(string)
	c.Server.TLS.Override = utils.Coalesce(overrides.Server.TLS.Override, c.Server.TLS.Override, overrides.Server.TLS.Override == nil).(*tls.Config)

	c.Server.TLS.CertFile = patchAbsFilePath(c.Server.TLS.CertFile, file)
	c.Server.TLS.KeyFile = patchAbsFilePath(c.Server.TLS.KeyFile, file)
}

// --- TIMEOUT
func (c *Configuration) copyOverWithTimeout(overrides Configuration, file *string) {
	c.Server.Timeout.Read = utils.Coalesce(overrides.Server.Timeout.Read, c.Server.Timeout.Read, overrides.Server.Timeout.Read == 0).(time.Duration)
	c.Server.Timeout.ReadHeader = utils.Coalesce(overrides.Server.Timeout.ReadHeader, c.Server.Timeout.ReadHeader, overrides.Server.Timeout.ReadHeader == 0).(time.Duration)
	c.Server.Timeout.Write = utils.Coalesce(overrides.Server.Timeout.Write, c.Server.Timeout.Write, overrides.Server.Timeout.Write == 0).(time.Duration)
	c.Server.Timeout.Idle = utils.Coalesce(overrides.Server.Timeout.Idle, c.Server.Timeout.Idle, overrides.Server.Timeout.Idle == 0).(time.Duration)
	c.Server.Timeout.Handler = utils.Coalesce(overrides.Server.Timeout.Handler, c.Server.Timeout.Handler, overrides.Server.Timeout.Handler == 0).(time.Duration)
}

// --- UPSTREAM
func (c *Configuration) copyOverWithUpstream(overrides Configuration, file *string) {
	c.Server.Upstream.Host = utils.Coalesce(overrides.Server.Upstream.Host, c.Server.Upstream.Host, overrides.Server.Upstream.Host == "").(string)
	c.Server.Upstream.Port = utils.Coalesce(overrides.Server.Upstream.Port, c.Server.Upstream.Port, overrides.Server.Upstream.Port == "").(string)
	c.Server.Upstream.Scheme = utils.Coalesce(overrides.Server.Upstream.Scheme, c.Server.Upstream.Scheme, overrides.Server.Upstream.Scheme == "").(string)
	c.Server.Upstream.Endpoints = utils.Coalesce(overrides.Server.Upstream.Endpoints, c.Server.Upstream.Endpoints, len(overrides.Server.Upstream.Endpoints) == 0).([]string)
	c.Server.Upstream.HTTP2HTTPS = utils.Coalesce(overrides.Server.Upstream.HTTP2HTTPS, c.Server.Upstream.HTTP2HTTPS, !overrides.Server.Upstream.HTTP2HTTPS).(bool)
	c.Server.Upstream.InsecureBridge = utils.Coalesce(overrides.Server.Upstream.InsecureBridge, c.Server.Upstream.InsecureBridge, !overrides.Server.Upstream.InsecureBridge).(bool)
	c.Server.Upstream.RedirectStatusCode = utils.Coalesce(overrides.Server.Upstream.RedirectStatusCode, c.Server.Upstream.RedirectStatusCode, overrides.Server.Upstream.RedirectStatusCode == 0).(int)
}

// --- CACHE
func (c *Configuration) copyOverWithCache(overrides Configuration, file *string) {
	c.Cache.Host = utils.Coalesce(overrides.Cache.Host, c.Cache.Host, overrides.Cache.Host == "").(string)
	c.Cache.Port = utils.Coalesce(overrides.Cache.Port, c.Cache.Port, overrides.Cache.Port == "").(string)
	c.Cache.Password = utils.Coalesce(overrides.Cache.Password, c.Cache.Password, overrides.Cache.Password == "").(string)
	c.Cache.DB = utils.Coalesce(overrides.Cache.DB, c.Cache.DB, overrides.Cache.DB == 0).(int)
	c.Cache.TTL = utils.Coalesce(overrides.Cache.TTL, c.Cache.TTL, overrides.Cache.TTL == 0).(int)
	c.Cache.AllowedStatuses = utils.Coalesce(overrides.Cache.AllowedStatuses, c.Cache.AllowedStatuses, len(overrides.Cache.AllowedStatuses) == 0 || overrides.Cache.AllowedStatuses[0] == 0).([]int)
	c.Cache.AllowedMethods = utils.Coalesce(overrides.Cache.AllowedMethods, c.Cache.AllowedMethods, len(overrides.Cache.AllowedMethods) == 0 || overrides.Cache.AllowedMethods[0] == "").([]string)

	c.Cache.AllowedMethods = append(c.Cache.AllowedMethods, "HEAD", "GET")
	c.Cache.AllowedMethods = slice.Unique(c.Cache.AllowedMethods)
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
func GetDomains() []DomainSet {
	domains := make(map[string]DomainSet)

	// add global upstream server...
	domains[Config.Server.Upstream.Host+utils.StringSeparatorOne+Config.Server.Upstream.Scheme] = DomainSet{
		Host:   Config.Server.Upstream.Host,
		Scheme: Config.Server.Upstream.Scheme,
	}

	for _, v := range Config.Domains {
		domains[v.Server.Upstream.Host+utils.StringSeparatorOne+v.Server.Upstream.Scheme] = DomainSet{
			Host:   v.Server.Upstream.Host,
			Scheme: v.Server.Upstream.Scheme,
		}
	}

	domainsUnique := make([]DomainSet, 0, len(domains))
	for _, d := range domains {
		domainsUnique = append(domainsUnique, d)
	}

	return domainsUnique
}

// DomainConf - Returns the configuration for the requested domain.
func DomainConf(domain string, scheme string) *Configuration {
	// Memoization
	if domainsCache == nil {
		domainsCache = make(map[string]*Configuration)
	}
	keyCache := fmt.Sprintf("%s%s%s", domain, utils.StringSeparatorOne, scheme)
	if val, ok := domainsCache[keyCache]; ok {
		log.Debugf("Cached configuration for %s", keyCache)
		return val
	}

	domainsCache[keyCache] = domainConfLookup(domain, scheme)
	return domainsCache[keyCache]
}

func domainConfLookup(domain string, scheme string) *Configuration {
	cleanedDomain := utils.StripPort(domain)

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
