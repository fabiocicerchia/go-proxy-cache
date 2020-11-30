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
	"time"

	"github.com/NYTimes/gziphandler"
	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
)

// CreateServerConfig - Generates the http.Server configuration.
func CreateServerConfig(domain string, port string) *http.Server {
	// TODO: THIS IS FOR EVERY DOMAIN, NO DOMAIN OVERRIDE. ACCEPTS AS LIMITATION (CREATE AN ISSUE)
	timeout := config.Config.Server.Timeout
	gzip := config.Config.Server.GZip

	mux := http.NewServeMux()

	// handlers
	mux.HandleFunc("/healthcheck", handler.HandleHealthcheck)
	mux.HandleFunc("/", handler.HandleRequest)

	muxWithMiddlewares := http.TimeoutHandler(
		mux,
		timeout.Handler,
		"Timed Out\n",
	)

	if gzip {
		// TODO: COVERAGE
		muxWithMiddlewares = gziphandler.GzipHandler(muxWithMiddlewares)
	}

	// TODO: TEST timeouts with custom handlers
	server := &http.Server{
		Addr:              ":" + port,
		ReadTimeout:       time.Duration(timeout.Read) * time.Second,
		WriteTimeout:      time.Duration(timeout.Write) * time.Second,
		IdleTimeout:       time.Duration(timeout.Idle) * time.Second,
		ReadHeaderTimeout: time.Duration(timeout.ReadHeader) * time.Second,
		Handler:           muxWithMiddlewares,
	}

	return server
}

// GetServerConfigs - Returns a http.Server configuration for HTTP and HTTPS.
func GetServerConfigs(domain string, domainConfig *config.Configuration) (*http.Server, *http.Server) {
	srvHTTP := CreateServerConfig(domain, domainConfig.Server.Port.HTTP)

	srvHTTPS := CreateServerConfig(domain, domainConfig.Server.Port.HTTPS)
	srvtls.ServerOverrides(domain, srvHTTPS, &srvtls.CertificatePair{
		Cert: domainConfig.Server.TLS.CertFile,
		Key:  domainConfig.Server.TLS.KeyFile,
	})

	return srvHTTP, srvHTTPS
}

// StartDomainServer - Configures and start listinening for a particular domain.
func StartDomainServer(domain string, serversHTTP map[string]*http.Server, serversHTTPS map[string]*http.Server) {
	domainConfig := config.DomainConf(domain)
	if domainConfig == nil {
		log.Errorf("Missing configuration for %s.", domain)
		return
	}

	// redis connect
	config.InitCircuitBreaker(domain, domainConfig.CircuitBreaker)
	engine.InitConn(domain, domainConfig.Cache)

	// Log setup values
	logger.LogSetup(domainConfig.Server)

	// config server http & https
	srvHTTP, srvHTTPS := GetServerConfigs(domain, domainConfig)

	// lb
	balancer.InitRoundRobin(domain, domainConfig.Server.Forwarding.Endpoints)

	serversHTTP[domainConfig.Server.Port.HTTP] = srvHTTP
	serversHTTPS[domainConfig.Server.Port.HTTPS] = srvHTTPS
}

// Start the GoProxyCache server.
func Start(configFile string) {
	// Init configs
	config.InitConfigFromFileOrEnv(configFile)
	config.Print()

	serversHTTP := make(map[string]*http.Server)
	serversHTTPS := make(map[string]*http.Server)
	for _, domain := range config.GetDomains() {
		StartDomainServer(domain, serversHTTP, serversHTTPS)
	}

	// start server http & https
	for _, srvHTTP := range serversHTTP {
		go func() { log.Fatal(srvHTTP.ListenAndServe()) }()
	}
	for _, srvHTTPS := range serversHTTPS {
		go func() { log.Fatal(srvHTTPS.ListenAndServeTLS("", "")) }()
	}

	// Wait for an interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// Attempt a graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for k, v := range serversHTTP {
		err := v.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Cannot shutdown server %s: %s", k, err)
		}
	}
	for k, v := range serversHTTPS {
		err := v.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Cannot shutdown server %s: %s", k, err)
		}
	}
}
