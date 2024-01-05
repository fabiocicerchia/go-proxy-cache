package config

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/sirupsen/logrus"
)

// DefaultTimeoutRead - Default value used for http.Server.ReadTimeout
var DefaultTimeoutRead time.Duration = 5 * time.Second

// DefaultTimeoutReadHeader - Default value used for http.Server.ReadHeaderTimeout
var DefaultTimeoutReadHeader time.Duration = 2 * time.Second

// DefaultTimeoutWrite - Default value used for http.Server.WriteTimeout
var DefaultTimeoutWrite time.Duration = 5 * time.Second

// DefaultTimeoutIdle - Default value used for http.Server.IdleTimeout
var DefaultTimeoutIdle time.Duration = 20 * time.Second

// DefaultTimeoutHandler - Default value used for http.TimeoutHandler
var DefaultTimeoutHandler time.Duration = 5 * time.Second

// DefaultCBThreshold - Default value used for circuitbreaker.CircuitBreaker.Threshold
var DefaultCBThreshold uint32 = 2

// DefaultCBFailureRate - Default value used for circuitbreaker.CircuitBreaker.FailureRate
var DefaultCBFailureRate float64 = 0.5

// DefaultCBInterval - Default value used for circuitbreaker.CircuitBreaker.Interval
var DefaultCBInterval time.Duration = 0 * time.Second

// DefaultCBTimeout - Default value used for circuitbreaker.CircuitBreaker.Timeout
var DefaultCBTimeout time.Duration = 60 * time.Second

// DefaultCBMaxRequests - Default value used for circuitbreaker.CircuitBreaker.MaxRequests
var DefaultCBMaxRequests uint32 = 1

// Configuration - Defines the server configuration.
type Configuration struct {
	Server         Server                        `yaml:"server"`
	Cache          Cache                         `yaml:"cache"`
	CircuitBreaker circuitbreaker.CircuitBreaker `yaml:"circuit_breaker"`
	Domains        Domains                       `yaml:"domains"`
	Log            Log                           `yaml:"log"`
	Tracing        Tracing                       `yaml:"tracing"`
	domainsCache   map[string]Configuration
	Jwt		   	   Jwt					 		 `yaml:"jwt"`
}

// Domains - Overrides per domain.
type Domains map[string]Configuration

// Server - Defines basic info for the server.
type Server struct {
	Port      Port      `yaml:"port"`
	TLS       TLS       `yaml:"tls"`
	Timeout   Timeout   `yaml:"timeout"`
	Upstream  Upstream  `yaml:"upstream"`
	GZip      bool      `yaml:"gzip" envconfig:"GZIP_ENABLED"`
	Internals Internals `yaml:"internals"`
}

// Port - Defines the listening ports per protocol.
type Port struct {
	HTTPS string `yaml:"https" envconfig:"SERVER_HTTPS_PORT"`
	HTTP  string `yaml:"http" envconfig:"SERVER_HTTP_PORT"`
}

// TLS - Defines the configuration for SSL/TLS.
type TLS struct {
	Auto     bool        `yaml:"auto" envconfig:"TLS_AUTO_CERT"`
	Email    string      `yaml:"email" envconfig:"TLS_EMAIL"`
	CertFile string      `yaml:"cert_file" envconfig:"TLS_CERT_FILE"`
	KeyFile  string      `yaml:"key_file" envconfig:"TLS_KEY_FILE"`
	Override *tls.Config `yaml:"override"`
}

// Upstream - Defines the upstream settings.
type Upstream struct {
	Host               string      `yaml:"host" envconfig:"FORWARD_HOST"`
	Port               string      `yaml:"port" envconfig:"FORWARD_PORT"`
	Scheme             string      `yaml:"scheme" envconfig:"FORWARD_SCHEME"`
	BalancingAlgorithm string      `yaml:"balancing_algorithm" envconfig:"BALANCING_ALGORITHM" default:"round-robin"`
	Endpoints          []string    `yaml:"endpoints" envconfig:"LB_ENDPOINT_LIST" split_words:"true"`
	InsecureBridge     bool        `yaml:"insecure_bridge"`
	HTTP2HTTPS         bool        `yaml:"http_to_https" envconfig:"HTTP2HTTPS"`
	RedirectStatusCode int         `yaml:"redirect_status_code" envconfig:"REDIRECT_STATUS_CODE" default:"301"`
	HealthCheck        HealthCheck `yaml:"health_check"`
}

// GetDomainID - Returns the unique ID for the upstream.
func (u Upstream) GetDomainID() string {
	return utils.IfEmpty(u.Host, "*") + utils.StringSeparatorOne + u.Scheme
}

// HealthCheck - Defines the health check settings.
type HealthCheck struct {
	StatusCodes   []string      `yaml:"status_codes" envconfig:"HEALTHCHECK_STATUS_CODES" split_words:"true"`
	Timeout       time.Duration `yaml:"timeout" envconfig:"HEALTHCHECK_TIMEOUT"`
	Interval      time.Duration `yaml:"interval" envconfig:"HEALTHCHECK_INTERVAL"`
	Port          string        `yaml:"port" envconfig:"HEALTHCHECK_PORT" default:"443"`
	Scheme        string        `yaml:"scheme" envconfig:"HEALTHCHECK_SCHEME" default:"https"`
	AllowInsecure bool          `yaml:"allow_insecure" envconfig:"HEALTHCHECK_ALLOW_INSECURE"`
}

// Timeout - Defines the server timeouts.
type Timeout struct {
	Read       time.Duration `yaml:"read" envconfig:"TIMEOUT_READ"`
	ReadHeader time.Duration `yaml:"read_header" envconfig:"TIMEOUT_READ_HEADER"`
	Write      time.Duration `yaml:"write" envconfig:"TIMEOUT_WRITE"`
	Idle       time.Duration `yaml:"idle" envconfig:"TIMEOUT_IDLE"`
	Handler    time.Duration `yaml:"handler" envconfig:"TIMEOUT_HANDLER"`
}

// Cache - Defines the config for the cache backend.
type Cache struct {
	Hosts           []string `yaml:"hosts" envconfig:"REDIS_HOSTS"`
	Password        string   `yaml:"password" envconfig:"REDIS_PASSWORD"`
	DB              int      `yaml:"db" envconfig:"REDIS_DB"`
	TTL             int      `yaml:"ttl" envconfig:"DEFAULT_TTL"`
	AllowedStatuses []int    `yaml:"allowed_statuses" envconfig:"CACHE_ALLOWED_STATUSES" split_words:"true"`
	AllowedMethods  []string `yaml:"allowed_methods" envconfig:"CACHE_ALLOWED_METHODS" split_words:"true"`
}

// Log - Defines the config for the logs.
type Log struct {
	TimeFormat     string `yaml:"time_format"`
	Format         string `yaml:"format"`
	SentryDsn      string `yaml:"sentry_dsn" envconfig:"SENTRY_DSN"`
	SyslogProtocol string `yaml:"syslog_protocol" envconfig:"SYSLOG_PROTOCOL"`
	SyslogEndpoint string `yaml:"syslog_endpoint" envconfig:"SYSLOG_ENDPOINT"`
}

// Tracing - Defines the config for the OpenTelemetry tracing.
type Tracing struct {
	JaegerEndpoint string  `yaml:"jaeger_endpoint" envconfig:"TRACING_JAEGER_ENDPOINT"`
	Enabled        bool    `yaml:"enabled" envconfig:"TRACING_ENABLED"`
	SamplingRatio  float64 `yaml:"sampling_ratio" envconfig:"TRACING_SAMPLING_RATIO" default:"1.0"`
}

// Internals - Defines the config for the internal listening address/port.
type Internals struct {
	ListeningAddress string `yaml:"listening_address" envconfig:"INTERNAL_LISTENING_ADDRESS" default:"127.0.0.1"`
	ListeningPort    string `yaml:"listening_port" envconfig:"INTERNAL_LISTENING_PORT" default:"52021"`
}

// DomainSet - Holds the uniqueness details of the domain.
type DomainSet struct {
	Host   string
	Scheme string
}

// Jwt - Defines the config for the jwt validation.
type Jwt struct {
	ExcludedPaths       []string   `yaml:"excluded_paths" envconfig:"JWT_EXCLUDED_PATHS" split_words:"true"`
	AllowedScopes       []string   `yaml:"allowed_scopes" envconfig:"JWT_ALLOWED_SCOPES" split_words:"true"`
	JwksUrl             string     `yaml:"jwks_url" envconfig:"JWT_JWKS_URL"`
	JwksRefreshInterval int        `yaml:"jwks_refresh_interval" envconfig:"JWT_REFRESH_INTERVAL" default:"15"`
	JwkCache            *jwk.Cache
	Context             context.Context
	Logger              *logrus.Logger
}

// Jwt - Defines the jwt validation error.
type JwtError struct {
	ErrorCode        string `json:"errorCode"`
	ErrorDescription string `json:"errorDescription"`
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
			BalancingAlgorithm: "round-robin",
			HealthCheck: HealthCheck{
				StatusCodes: []string{"200"},
				Scheme:      "https",
			},
		},
		GZip: false,
	},
	Cache: Cache{
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
