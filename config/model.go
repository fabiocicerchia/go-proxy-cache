//nolint: lll
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
	"net/http"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

var DefaultTimeoutRead time.Duration = 5 * time.Second
var DefaultTimeoutReadHeader time.Duration = 2 * time.Second
var DefaultTimeoutWrite time.Duration = 5 * time.Second
var DefaultTimeoutIdle time.Duration = 20 * time.Second
var DefaultTimeoutHandler time.Duration = 5 * time.Second
var DefaultCBThreshold uint32 = 2
var DefaultCBFailureRate float64 = 0.5
var DefaultCBInterval time.Duration = 0 * time.Second
var DefaultCBTimeout time.Duration = 60 * time.Second
var DefaultCBMaxRequests uint32 = 1

// Configuration - Defines the server configuration.
type Configuration struct {
	Server         Server                        `yaml:"server"`
	Cache          Cache                         `yaml:"cache"`
	CircuitBreaker circuitbreaker.CircuitBreaker `yaml:"circuit_breaker"`
	Domains        Domains                       `yaml:"domains"`
	Log            Log                           `yaml:"log"`
}

// Domains - Overrides per domain.
type Domains map[string]Configuration

// Server - Defines basic info for the server.
type Server struct {
	Port        Port     `yaml:"port"`
	TLS         TLS      `yaml:"tls"`
	Timeout     Timeout  `yaml:"timeout"`
	Upstream    Upstream `yaml:"upstream"`
	GZip        bool     `yaml:"gzip",envconfig:"GZIP_ENABLED"`
	Healthcheck bool     `yaml:"healthcheck"`
}

// Port - Defines the listening ports per protocol.
type Port struct {
	HTTPS string `yaml:"https",envconfig:"SERVER_HTTPS_PORT"`
	HTTP  string `yaml:"http",envconfig:"SERVER_HTTP_PORT"`
}

// TLS - Defines the configuration for SSL/TLS.
type TLS struct {
	Auto     bool        `yaml:"auto",envconfig:"TLS_AUTO_CERT"`
	Email    string      `yaml:"email",envconfig:"TLS_EMAIL"`
	CertFile string      `yaml:"cert_file",envconfig:"TLS_CERT_FILE"`
	KeyFile  string      `yaml:"key_file",envconfig:"TLS_KEY_FILE"`
	Override *tls.Config `yaml:"override"`
}

// Upstream - Defines the upstream settings.
type Upstream struct {
	Host               string   `yaml:"host",envconfig:"FORWARD_HOST"`
	Port               string   `yaml:"port",envconfig:"FORWARD_PORT"`
	Scheme             string   `yaml:"scheme",envconfig:"FORWARD_SCHEME"`
	Endpoints          []string `yaml:"endpoints",envconfig:"LB_ENDPOINT_LIST",split_words:"true"`
	InsecureBridge     bool     `yaml:"insecure_bridge"`
	HTTP2HTTPS         bool     `yaml:"http_to_https",envconfig:"HTTP2HTTPS"`
	RedirectStatusCode int      `yaml:"redirect_status_code",envconfig:"REDIRECT_STATUS_CODE"`
}

// GetDomainID - Returns the unique ID for the upstream.
func (u Upstream) GetDomainID() string {
	return utils.IfEmpty(u.Host, "*") + utils.StringSeparatorOne + u.Scheme
}

// Timeout - Defines the server timeouts.
type Timeout struct {
	Read       time.Duration `yaml:"read",envconfig:"TIMEOUT_READ"`
	ReadHeader time.Duration `yaml:"read_header",envconfig:"TIMEOUT_READ_HEADER"`
	Write      time.Duration `yaml:"write",envconfig:"TIMEOUT_WRITE"`
	Idle       time.Duration `yaml:"idle",envconfig:"TIMEOUT_IDLE"`
	Handler    time.Duration `yaml:"handler",envconfig:"TIMEOUT_HANDLER"`
}

// Cache - Defines the config for the cache backend.
type Cache struct {
	Host            string   `yaml:"host",envconfig:"REDIS_HOST"`
	Port            string   `yaml:"port",envconfig:"REDIS_PORT"`
	Password        string   `yaml:"password",envconfig:"REDIS_PASSWORD"`
	DB              int      `yaml:"db",envconfig:"REDIS_DB"`
	TTL             int      `yaml:"ttl",envconfig:"DEFAULT_TTL"`
	AllowedStatuses []int    `yaml:"allowed_statuses",envconfig:"CACHE_ALLOWED_STATUSES",split_words:"true"`
	AllowedMethods  []string `yaml:"allowed_methods",envconfig:"CACHE_ALLOWED_METHODS",split_words:"true"`
}

// Log - Defines the config for the logs.
type Log struct {
	TimeFormat string `yaml:"time_format"`
	Format     string `yaml:"format"`
}

// DomainSet - Holds the uniqueness details of the domain.
type DomainSet struct {
	Host   string
	Scheme string
}

// Config - Holds the server configuration.
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
			Read:       DefaultTimeoutRead,
			ReadHeader: DefaultTimeoutReadHeader,
			Write:      DefaultTimeoutWrite,
			Idle:       DefaultTimeoutIdle,
			Handler:    DefaultTimeoutHandler,
		},
		Upstream: Upstream{
			HTTP2HTTPS:         false,
			InsecureBridge:     false,
			RedirectStatusCode: http.StatusPermanentRedirect,
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
		Threshold:   DefaultCBThreshold,   // after 2nd request, if meet FailureRate goes open.
		FailureRate: DefaultCBFailureRate, // 1 out of 2 fails, or more
		Interval:    DefaultCBInterval,    // doesn't clears counts
		Timeout:     DefaultCBTimeout,     // clears state after 60s
		MaxRequests: DefaultCBMaxRequests,
	},
	Log: Log{
		TimeFormat: "2006/01/02 15:04:05",
		Format:     `$host - $remote_addr - $remote_user $protocol $request_method "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $cached_status_label`,
	},
}
