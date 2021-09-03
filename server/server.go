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
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
	"github.com/fabiocicerchia/go-proxy-cache/utils/queue"
)

const enableTimeoutHandler = true

// DefaultTimeoutShutdown - Default Timeout for shutting down a context.
const DefaultTimeoutShutdown time.Duration = 5 * time.Second

type Server struct {
	Domain  string
	HttpSrv http.Server
}

// Servers - Contains the HTTP/HTTPS servers.
type Servers struct {
	HTTP  map[string]Server
	HTTPS map[string]Server
}

// Run - Starts the GoProxyCache servers' listeners.
func Run(configFile string) {
	log.Infof("Starting...\n")

	// Init configs
	config.InitConfigFromFileOrEnv(configFile)
	config.Print()

	servers := &Servers{
		HTTP:  make(map[string]Server),
		HTTPS: make(map[string]Server),
	}

	for _, domain := range config.GetDomains() {
		servers.StartDomainServer(domain.Host, domain.Scheme)
	}

	// init queue
	queue.Init()

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

	log.Error("Finishing processing queue...")
	queue.Init()

	log.Error("Shutting down servers...")
	servers.shutdownServers(ctx)

	log.Error("All listeners shut down. Exiting.")
}

// InitServer - Generates the http.Server configuration.
func InitServer(domain string, domainConfig config.Configuration) http.Server {
	mux := http.NewServeMux()

	// handlers
	if domainConfig.Server.Healthcheck {
		mux.HandleFunc("/healthcheck", handler.HandleHealthcheck(domainConfig))
	}

	mux.HandleFunc("/", handler.HandleRequest)

	// basic
	var muxMiddleware http.Handler = mux

	// timeout middleware
	// NOTE: THIS IS FOR EVERY DOMAIN, NO DOMAIN OVERRIDE.
	//       WHEN SHARING SAME PORT NO CUSTOM OVERRIDES ON CRITICAL SETTINGS.
	// TODO! CONVERT FOR DOMAIN CONFIG
	timeout := domainConfig.Server.Timeout
	if enableTimeoutHandler && timeout.Handler > 0 {
		muxMiddleware = http.TimeoutHandler(muxMiddleware, timeout.Handler, "Timed Out\n")
	}

	server := http.Server{
		ReadTimeout:       timeout.Read * time.Second,
		WriteTimeout:      timeout.Write * time.Second,
		IdleTimeout:       timeout.Idle * time.Second,
		ReadHeaderTimeout: timeout.ReadHeader * time.Second,
		Handler:           muxMiddleware,
	}

	return server
}

// AttachPlain - Adds a new HTTP server in the listener container.
// NOTE: There will be only ONE server listening on a port.
//       This means the last processed will override all the previous shared
//       settings. THIS COULD LEAD TO CONFLICTS WHEN SHARING THE SAME PORT.
func (s *Servers) AttachPlain(domain string, port string, server http.Server) {
	s.HTTP[port] = Server{Domain: domain, HttpSrv: server}
}

// AttachSecure - Adds a new HTTPS server in the listener container.
// NOTE: There will be only ONE server listening on a port.
//       This means the last processed will override all the previous shared
//       settings. THIS COULD LEAD TO CONFLICTS WHEN SHARING THE SAME PORT.
func (s *Servers) AttachSecure(domain string, port string, server http.Server) {
	s.HTTPS[port] = Server{Domain: domain, HttpSrv: server}
}

// InitServers - Returns a http.Server configuration for HTTP and HTTPS.
func (s *Servers) InitServers(domain string, domainConfig config.Configuration) {
	srvHTTP := InitServer(domain, domainConfig)
	s.AttachPlain(domain, domainConfig.Server.Port.HTTP, srvHTTP)

	srvHTTPS := InitServer(domain, domainConfig)

	err := srvtls.ServerOverrides(domain, &srvHTTPS, domainConfig.Server)
	if err != nil {
		log.Errorf("Skipping '%s' TLS server configuration: %s", domain, err)
		log.Errorf("No HTTPS server will be listening on '%s'", domain)

		return
	}

	s.AttachSecure(domain, domainConfig.Server.Port.HTTPS, srvHTTPS)
}

// StartDomainServer - Configures and start listening for a particular domain.
func (s *Servers) StartDomainServer(domain string, scheme string) {
	domainConfig, found := config.DomainConf(domain, scheme)
	if !found {
		log.Errorf("Missing configuration for %s.", domain)
		return
	}

	domainID := domainConfig.Server.Upstream.GetDomainID()

	// Log setup values
	logger.LogSetup(domainConfig.Server)

	// redis connect
	circuitbreaker.InitCircuitBreaker(domainID, domainConfig.CircuitBreaker)
	engine.InitConn(domainID, domainConfig.Cache, log.StandardLogger())

	// config server http & https
	s.InitServers(domain, domainConfig)

	// lb
	balancer.InitRoundRobin(domainID, domainConfig.Server.Upstream.Endpoints)
}

func (s Servers) startListeners() {
	for port, srvHTTP := range s.HTTP {
		l, err := net.Listen("tcp", ":"+port)
		if err != nil {
			panic(err)
		}

		go func(srv http.Server, l net.Listener) {
			log.Fatal(srv.Serve(l))
		}(srvHTTP.HttpSrv, l)
	}

	for port, srvHTTPS := range s.HTTPS {
		l, err := net.Listen("tcp", ":"+port)
		if err != nil {
			panic(err)
		}

		go func(srv http.Server, l net.Listener) {
			log.Fatal(srv.ServeTLS(l, "", ""))
		}(srvHTTPS.HttpSrv, l)
	}
}

func (s Servers) shutdownServers(ctx context.Context) {
	for k, v := range s.HTTP {
		err := v.HttpSrv.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Cannot shutdown server %s: %s", k, err)
		}
	}

	for k, v := range s.HTTPS {
		err := v.HttpSrv.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Cannot shutdown server %s: %s", k, err)
		}
	}
}
