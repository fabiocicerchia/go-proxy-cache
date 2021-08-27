package server

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NYTimes/gziphandler"
	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

const enableTimeoutHandler = true

// DefaultTimeoutShutdown - Default Timeout for shutting down a context.
const DefaultTimeoutShutdown time.Duration = 5 * time.Second

// Servers - Contains the HTTP/HTTPS servers.
type Servers struct {
	HTTP  map[string]*http.Server
	HTTPS map[string]*http.Server
}

// Run - Starts the GoProxyCache servers' listeners.
func Run(configFile string) {
	log.Infof("Starting...\n")

	// Init configs
	config.InitConfigFromFileOrEnv(configFile)
	config.Print()

	servers := &Servers{
		HTTP:  make(map[string]*http.Server),
		HTTPS: make(map[string]*http.Server),
	}

	for _, domain := range config.GetDomains() {
		servers.StartDomainServer(domain.Host, domain.Scheme)
	}

	// start server http & https
	servers.startListeners()

	log.Infof("Waiting for incoming connections...\n")

	// Wait for an interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Error("SIGTERM or SIGINT caught, shutting down...")

	// Attempt a graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeoutShutdown)
	defer cancel()

	servers.shutdownServers(ctx)
	log.Error("all listeners shut down.")
}

// InitServer - Generates the http.Server configuration.
func InitServer(domain string) *http.Server {
	// THIS IS FOR EVERY DOMAIN, NO DOMAIN OVERRIDE.
	timeout := config.Config.Server.Timeout
	gzip := config.Config.Server.GZip

	mux := http.NewServeMux()

	// handlers
	if config.Config.Server.Healthcheck {
		mux.HandleFunc("/healthcheck", handler.HandleHealthcheck)
	}

	mux.HandleFunc("/", handler.HandleRequest)

	// basic
	var muxMiddleware http.Handler = mux

	// etag middleware
	muxMiddleware = ConditionalETag(muxMiddleware)

	// gzip middleware
	if gzip {
		muxMiddleware = gziphandler.GzipHandler(muxMiddleware)
	}

	// timeout middleware
	if enableTimeoutHandler && timeout.Handler > 0 {
		muxMiddleware = http.TimeoutHandler(muxMiddleware, timeout.Handler, "Timed Out\n")
	}

	server := &http.Server{
		ReadTimeout:       timeout.Read * time.Second,
		WriteTimeout:      timeout.Write * time.Second,
		IdleTimeout:       timeout.Idle * time.Second,
		ReadHeaderTimeout: timeout.ReadHeader * time.Second,
		Handler:           muxMiddleware,
	}

	return server
}

// AttachPlain - Adds a new HTTP server in the listener container.
func (s *Servers) AttachPlain(port string, server *http.Server) {
	s.HTTP[port] = server
	s.HTTP[port].Addr = ":" + port
}

// AttachSecure - Adds a new HTTPS server in the listener container.
func (s *Servers) AttachSecure(port string, server *http.Server) {
	s.HTTPS[port] = server
	s.HTTPS[port].Addr = ":" + port
}

// InitServers - Returns a http.Server configuration for HTTP and HTTPS.
func (s *Servers) InitServers(domain string, domainConfig config.Server) {
	srv := InitServer(domain)
	s.AttachPlain(domainConfig.Port.HTTP, srv)

	srvHTTPS := InitServer(domain)

	err := srvtls.ServerOverrides(domain, srvHTTPS, domainConfig)
	if err != nil {
		log.Errorf("Skipping '%s' TLS server configuration: %s", domain, err)
		log.Errorf("No HTTPS server will be listening on '%s'", domain)

		return
	}

	s.AttachSecure(domainConfig.Port.HTTPS, srvHTTPS)
}

// StartDomainServer - Configures and start listening for a particular domain.
func (s *Servers) StartDomainServer(domain string, scheme string) {
	domainConfig := config.DomainConf(domain, scheme)
	if domainConfig == nil {
		log.Errorf("Missing configuration for %s.", domain)
		return
	}

	domainID := domainConfig.Server.Upstream.GetDomainID()

	// redis connect
	circuitbreaker.InitCircuitBreaker(domainID, domainConfig.CircuitBreaker)
	engine.InitConn(domainID, domainConfig.Cache)

	// Log setup values
	logger.LogSetup(domainConfig.Server)

	// config server http & https
	s.InitServers(domain, domainConfig.Server)

	// lb
	balancer.InitRoundRobin(domainID, domainConfig.Server.Upstream.Endpoints)
}

func (s Servers) startListeners() {
	for _, srvHTTP := range s.HTTP {
		go func(srv *http.Server) { log.Fatal(srv.ListenAndServe()) }(srvHTTP)
	}

	for _, srvHTTPS := range s.HTTPS {
		go func(srv *http.Server) { log.Fatal(srv.ListenAndServeTLS("", "")) }(srvHTTPS)
	}
}

func (s Servers) shutdownServers(ctx context.Context) {
	for k, v := range s.HTTP {
		err := v.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Cannot shutdown server %s: %s", k, err)
		}
	}

	for k, v := range s.HTTPS {
		err := v.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Cannot shutdown server %s: %s", k, err)
		}
	}
}
