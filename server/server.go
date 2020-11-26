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
	// THIS IS FOR EVERY DOMAIN, NO DOMAIN OVERRIDE. ACCEPTS AS LIMITATION (CREATE AN ISSUE)
	timeout := config.Config.Server.Timeout
	gzip := config.Config.Server.GZip

	mux := http.NewServeMux()

	// handlers
	mux.HandleFunc("/healthcheck", handler.HandleHealthcheck)
	mux.HandleFunc("/", handler.HandleRequest)

	muxWithMiddlewares := http.TimeoutHandler(
		mux,
		time.Duration(timeout.Handler)*time.Second,
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

func GetServerConfigs(domain string, domainConfig *config.Configuration) (*http.Server, *http.Server) {
	srvHTTP := CreateServerConfig(domain, domainConfig.Server.Port.HTTP)

	srvHTTPS := CreateServerConfig(domain, domainConfig.Server.Port.HTTPS)
	srvtls.ServerOverrides(domain, srvHTTPS, &srvtls.CertificatePair{
		Cert: domainConfig.Server.TLS.CertFile,
		Key:  domainConfig.Server.TLS.KeyFile,
	})

	return srvHTTP, srvHTTPS
}

func StartDomainServer(domain string, servers map[string]*http.Server) {
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

	// start server http & https
	if _, ok := servers[domainConfig.Server.Port.HTTP]; !ok {
		servers[domainConfig.Server.Port.HTTP] = srvHTTP

		go func() { log.Fatal(srvHTTP.ListenAndServe()) }()
	}
	if _, ok := servers[domainConfig.Server.Port.HTTPS]; !ok {
		servers[domainConfig.Server.Port.HTTPS] = srvHTTPS

		go func() { log.Fatal(srvHTTPS.ListenAndServeTLS("", "")) }()
	}

	// lb
	balancer.InitRoundRobin(domain, domainConfig.Server.Forwarding.Endpoints)
}

// Start the GoProxyCache server.
func Start() {
	// Init configs
	config.InitConfigFromFileOrEnv("config.yml")
	config.Print()

	servers := make(map[string]*http.Server)
	for _, domain := range config.GetDomains() {
		StartDomainServer(domain, servers)
	}

	// Wait for an interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// Attempt a graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for k, v := range servers {
		err := v.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Cannot shutdown server %s: %s", k, err)
		}
	}
}
