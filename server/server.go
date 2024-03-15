package server

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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/jwt"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

const enableTimeoutHandler = true

// DefaultTimeoutShutdown - Default Timeout for shutting down a context.
const DefaultTimeoutShutdown time.Duration = 5 * time.Second

// Server - Contains the core info about an HTTP server.
type Server struct {
	Domain  string
	HttpSrv *http.Server
}

// Servers - Contains the HTTP/HTTPS servers.
type Servers struct {
	HTTP  map[string]*Server
	HTTPS map[string]*Server
}

var servers *Servers

// Run - Starts the GoProxyCache servers' listeners.
func Run(appVersion string, configFile string) {
	log.Infof("Starting...\n")

	ctx := context.Background()

	// Init configs
	config.InitConfigFromFileOrEnv(configFile)
	config.Print()

	// Logging Hooks
	log := logger.GetGlobal()
	logger.HookSentry(log, config.Config.Log.SentryDsn)
	logger.HookSyslog(log, config.Config.Log.SyslogProtocol, config.Config.Log.SyslogEndpoint)

	// Init tracing
	if config.Config.Tracing.Enabled {
		tracerProvider, err := tracing.NewJaegerProvider(
			appVersion,
			config.Config.Tracing.JaegerEndpoint,
			config.Config.Tracing.SamplingRatio,
		)
		if err != nil {
			log.Fatalln(err)
		}

		// Register the TraceContext propagator globally.
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	}

	// init servers
	servers = &Servers{
		HTTP:  make(map[string]*Server),
		HTTPS: make(map[string]*Server),
	}

	for _, domain := range config.GetDomains() {
		servers.StartDomainServer(domain.Host, domain.Scheme)
	}
	servers.AttachPlain(
		config.Config.Server.Internals.ListeningAddress,
		config.Config.Server.Internals.ListeningPort,
		InitInternals(),
	)

	// start server http & https
	servers.startListeners()

	log.Infof("Waiting for incoming connections...\n")

	// Wait for an interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Error("SIGTERM or SIGINT caught, shutting down...")

	// Attempt a graceful shutdown
	ctxDown, cancel := context.WithTimeout(ctx, DefaultTimeoutShutdown)
	defer cancel()

	log.Error("Shutting down servers...")
	servers.shutdownServers(ctxDown)

	log.Error("All listeners shut down. Exiting.")
}

// InitInternals - Generates the internals endpoints (not exposed on public ports :80 :443).
func InitInternals() *http.Server {
	mux := http.NewServeMux()

	// handlers
	mux.HandleFunc("/healthcheck", handler.HandleHealthcheck(config.Config))

	metrics.Register()
	mux.Handle("/metrics", promhttp.Handler())

	timeout := config.Config.Server.Timeout

	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: timeout.ReadHeader * time.Second,
	}
}

// InitServer - Generates the http.Server configuration.
func InitServer(domain string, domainConfig config.Configuration) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", tracing.HTTPHandlerFunc(handler.HandleRequest, "handle_request"))

	// basic
	var muxMiddleware http.Handler = mux

	// timeout middleware
	// NOTE: THIS IS FOR EVERY DOMAIN, NO DOMAIN OVERRIDE.
	//       WHEN SHARING SAME PORT NO CUSTOM OVERRIDES ON CRITICAL SETTINGS.
	timeout := config.Config.Server.Timeout
	if enableTimeoutHandler && timeout.Handler > 0 {
		muxMiddleware = http.TimeoutHandler(muxMiddleware, timeout.Handler, "Timed Out\n")
	}

	server := &http.Server{
		ReadTimeout:       timeout.Read * time.Second,
		WriteTimeout:      timeout.Write * time.Second,
		IdleTimeout:       timeout.Idle * time.Second,
		ReadHeaderTimeout: timeout.ReadHeader * time.Second,
		Handler:           jwt.JWTHandler(muxMiddleware),
	}

	return server
}

// AttachPlain - Adds a new HTTP server in the listener container.
// NOTE: There will be only ONE server listening on a port.
//
//	This means the last processed will override all the previous shared
//	settings. THIS COULD LEAD TO CONFLICTS WHEN SHARING THE SAME PORT.
func (s *Servers) AttachPlain(domain string, port string, server *http.Server) {
	s.HTTP[port] = &Server{Domain: domain, HttpSrv: server}
}

// AttachSecure - Adds a new HTTPS server in the listener container.
// NOTE: There will be only ONE server listening on a port.
//
//	This means the last processed will override all the previous shared
//	settings. THIS COULD LEAD TO CONFLICTS WHEN SHARING THE SAME PORT.
func (s *Servers) AttachSecure(domain string, port string, server *http.Server) {
	s.HTTPS[port] = &Server{Domain: domain, HttpSrv: server}
}

// InitServers - Returns a http.Server configuration for HTTP and HTTPS.
func (s *Servers) InitServers(domain string, domainConfig config.Configuration) {
	srvHTTP := InitServer(domain, domainConfig)
	s.AttachPlain(domain, domainConfig.Server.Port.HTTP, srvHTTP)

	srvHTTPS := InitServer(domain, domainConfig)

	err := srvtls.ServerOverrides(domain, srvHTTPS, domainConfig.Server)
	if err != nil {
		logger.GetGlobal().Errorf("Skipping '%s' TLS server configuration: %s", domain, err)
		logger.GetGlobal().Errorf("No HTTPS server will be listening on '%s'", domain)

		return
	}

	s.AttachSecure(domain, domainConfig.Server.Port.HTTPS, srvHTTPS)
}

// StartDomainServer - Configures and start listening for a particular domain.
func (s *Servers) StartDomainServer(domain string, scheme string) {
	domainConfig, found := config.DomainConf(domain, scheme)
	if !found {
		logger.GetGlobal().Errorf("Missing configuration for %s.", domain)
		return
	}

	domainID := domainConfig.Server.Upstream.GetDomainID()

	// Log setup values
	logger.LogSetup(domainConfig.Server)

	// redis connect
	circuitbreaker.InitCircuitBreaker(domainID, domainConfig.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, domainConfig.Cache, logger.GetGlobal())

	// config server http & https
	s.InitServers(domain, domainConfig)

	// lb
	balancer.Init(domainID, domainConfig.Server.Upstream)
}

func (s Servers) startListeners() {
	for port, srvHTTP := range s.HTTP {
		srvHTTP.HttpSrv.Addr = ":" + port

		go func(srv *http.Server) {
			logger.GetGlobal().Fatal(srv.ListenAndServe())
		}(srvHTTP.HttpSrv)
	}

	for port, srvHTTPS := range s.HTTPS {
		srvHTTPS.HttpSrv.Addr = ":" + port

		go func(srv *http.Server) {
			logger.GetGlobal().Fatal(srv.ListenAndServeTLS("", ""))
		}(srvHTTPS.HttpSrv)
	}
}

func (s Servers) shutdownServers(ctx context.Context) {
	for k, v := range s.HTTP {
		err := v.HttpSrv.Shutdown(ctx)
		if err != nil {
			logger.GetGlobal().Fatalf("Cannot shutdown server %s: %s", k, err)
		}
	}

	for k, v := range s.HTTPS {
		err := v.HttpSrv.Shutdown(ctx)
		if err != nil {
			logger.GetGlobal().Fatalf("Cannot shutdown server %s: %s", k, err)
		}
	}
}
