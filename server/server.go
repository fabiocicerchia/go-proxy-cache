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
	"errors"
	"fmt"
	"net"
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

// normalizeTimeout - Returns a timeout ready to be used on an http.Server.
// Timeout values are time.Durations (e.g. TIMEOUT_READ=5s, as documented in
// .env.dist and config.yml.dist), so they must NOT be multiplied by
// time.Second again: 5s * time.Second overflows into ~158 years, which
// effectively disabled every server timeout (slowloris exposure).
// Sub-millisecond values are interpreted as bare seconds for backward
// compatibility with YAML configs that provided plain integers (e.g.
// `read: 5` parses as 5ns but clearly means 5s).
func normalizeTimeout(d time.Duration) time.Duration {
	if d > 0 && d < time.Millisecond {
		// d is a bare count (5ns == "5"), scale it to seconds.
		return d * time.Second
	}

	return d
}

// Server - Contains the core info about an HTTP server.
type Server struct {
	Domain   string
	HttpSrv  *http.Server
	Listener net.Listener
}

// Servers - Contains the HTTP/HTTPS servers.
type Servers struct {
	HTTP  map[string]*Server
	HTTPS map[string]*Server

	// inherited - Listening sockets handed to the process, by port (see listeners.go).
	inherited map[string]net.Listener
	conns     connStates
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

	inherited, err := inheritedListeners()
	if err != nil {
		log.Fatalln(err)
	}

	// init servers
	servers = &Servers{
		HTTP:      make(map[string]*Server),
		HTTPS:     make(map[string]*Server),
		inherited: inherited,
	}

	for _, domain := range config.GetDomains() {
		servers.StartDomainServer(domain.Host, domain.Scheme)
	}
	err = servers.AttachInternals(
		config.Config.Server.Internals.ListeningAddress,
		config.Config.Server.Internals.ListeningPort,
		InitInternals(),
	)
	if err != nil {
		log.Fatalln(err)
	}

	// start server http & https
	servers.startListeners()

	log.Infof("Waiting for incoming connections...\n")

	servers.waitForSignal(ctx, normalizeTimeout(config.Config.Server.Timeout.Shutdown))
}

// waitForSignal - Blocks until SIGINT/SIGTERM (drain, then return) or SIGHUP
// (drain, then re-execute in place, keeping the listening sockets).
func (s *Servers) waitForSignal(ctx context.Context, drain time.Duration) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-c

	// Taken before the drain, which closes the listeners: these copies keep the
	// sockets open, so connections queue in the kernel instead of being refused.
	var files []*os.File
	if sig == syscall.SIGHUP {
		var err error
		if files, err = s.listenerFiles(); err != nil {
			logger.GetGlobal().Errorf("Cannot hand the listeners over, exiting instead: %s", err)
		}
	}

	logger.GetGlobal().Errorf("%s caught, draining in-flight requests for up to %s...", sig, drain)

	ctxDown, cancel := context.WithTimeout(ctx, drain)
	defer cancel()

	s.shutdownServers(ctxDown)

	if files != nil {
		logger.GetGlobal().Fatalf("Cannot re-execute: %s", reexec(files))
	}

	logger.GetGlobal().Error("All listeners shut down. Exiting.")
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
		ReadHeaderTimeout: normalizeTimeout(timeout.ReadHeader),
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
		ReadTimeout:       normalizeTimeout(timeout.Read),
		WriteTimeout:      normalizeTimeout(timeout.Write),
		IdleTimeout:       normalizeTimeout(timeout.Idle),
		ReadHeaderTimeout: normalizeTimeout(timeout.ReadHeader),
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
	server.Addr = ":" + port
	s.HTTP[port] = &Server{Domain: domain, HttpSrv: server}
}

// AttachInternals - Adds the internal (healthcheck/metrics) server, bound to
// the configured address only. It is keyed by host:port so it can never
// replace a public listener, and sharing a public port is rejected because
// the internal endpoints would be either lost or exposed publicly.
func (s *Servers) AttachInternals(address string, port string, server *http.Server) error {
	if _, ok := s.HTTP[port]; ok {
		return fmt.Errorf("internal listening port %s is already used by a public HTTP listener", port)
	}
	if _, ok := s.HTTPS[port]; ok {
		return fmt.Errorf("internal listening port %s is already used by a public HTTPS listener", port)
	}

	server.Addr = net.JoinHostPort(address, port)
	s.HTTP[server.Addr] = &Server{Domain: address, HttpSrv: server}

	return nil
}

// AttachSecure - Adds a new HTTPS server in the listener container.
// NOTE: There will be only ONE server listening on a port.
//
//	This means the last processed will override all the previous shared
//	settings. THIS COULD LEAD TO CONFLICTS WHEN SHARING THE SAME PORT.
func (s *Servers) AttachSecure(domain string, port string, server *http.Server) {
	server.Addr = ":" + port
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

func (s *Servers) startListeners() {
	for port, srvHTTP := range s.HTTP {
		srvHTTP.HttpSrv.Addr = ":" + port
		srvHTTP.Listener = s.listen(port)
		srvHTTP.HttpSrv.ConnState = s.conns.track

		go func(srv *http.Server, ln net.Listener) {
			// ErrServerClosed/ErrClosed are returned on graceful shutdown and must
			// not kill the process (Fatal calls os.Exit and skips the drain).
			if err := srv.Serve(ln); !isClosed(err) {
				logger.GetGlobal().Fatal(err)
			}
		}(srvHTTP.HttpSrv, srvHTTP.Listener)
	}

	for port, srvHTTPS := range s.HTTPS {
		srvHTTPS.HttpSrv.Addr = ":" + port
		srvHTTPS.Listener = s.listen(port)
		srvHTTPS.HttpSrv.ConnState = s.conns.track

		go func(srv *http.Server, ln net.Listener) {
			if err := srv.ServeTLS(ln, "", ""); !isClosed(err) {
				logger.GetGlobal().Fatal(err)
			}
		}(srvHTTPS.HttpSrv, srvHTTPS.Listener)
	}

	for port, ln := range s.inherited {
		logger.GetGlobal().Warnf("Closing the inherited listener on port %s: no server is configured on it", port)
		_ = ln.Close()
	}
	s.inherited = nil
}

// listen - Returns the inherited listener for the port, or binds a new one.
func (s *Servers) listen(port string) net.Listener {
	if ln, ok := s.inherited[port]; ok {
		delete(s.inherited, port)
		logger.GetGlobal().Infof("Using the inherited listener on port %s", port)

		return ln
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.GetGlobal().Fatal(err)
	}

	return ln
}

func isClosed(err error) bool {
	return err == nil || errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed)
}

// shutdownServers - Stops accepting, lets every request already accepted finish,
// then shuts the servers down. Shutdown alone would drop a request on a
// just-accepted connection that it reads after starting (net/http conn.serve).
func (s *Servers) shutdownServers(ctx context.Context) {
	for _, group := range []map[string]*Server{s.HTTP, s.HTTPS} {
		for _, v := range group {
			_ = v.Listener.Close()
			v.HttpSrv.SetKeepAlivesEnabled(false) // closes idle conns, and the rest after their response
		}
	}

	for s.conns.busy() && ctx.Err() == nil {
		time.Sleep(10 * time.Millisecond)
	}

	// Log-and-continue: a Fatalf here would os.Exit on the first failing
	// server and skip draining all the remaining listeners.
	for _, group := range []map[string]*Server{s.HTTP, s.HTTPS} {
		for k, v := range group {
			if err := v.HttpSrv.Shutdown(ctx); !isClosed(err) {
				logger.GetGlobal().Errorf("Cannot shutdown server %s: %s", k, err)
			}
		}
	}
}
