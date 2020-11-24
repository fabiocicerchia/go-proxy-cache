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
func CreateServerConfig(
	domain string,
	port string,
	httpsPort string,
	certPair *srvtls.CertificatePair,
) *http.Server {
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
		muxWithMiddlewares = gziphandler.GzipHandler(muxWithMiddlewares) // TODO: NEEDS TO HANDLE DOMAINS
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

	if port == httpsPort { // TODO: NEEDS TO HANDLE DOMAINS
		srvtls.ServerOverrides(domain, server, certPair)
	}

	return server
}

// TODO: SPLIT THE HELL FROM THIS MESS.

// Start the GoProxyCache server.
func Start() {
	// Init configs
	config.InitConfigFromFileOrEnv("config.yml")
	config.Print()

	// redis connect
	config.InitCircuitBreaker(config.Config.CircuitBreaker)
	// TODO: ALLOW CUSTOM REDIS PER DOMAIN + DEDICATED CB PER REDIS SERVER WITH PREFIX
	engine.InitConn("global", config.Config.Cache)

	domains := config.GetDomains()

	serversHTTP := make(map[string]*http.Server)
	serversHTTPS := make(map[string]*http.Server)

	// TODO: use go routine.
	for _, domain := range domains {
		domainConfig := config.DomainConf(domain)
		if domainConfig == nil {
			log.Errorf("Missing configuration for %s.", domain)
			continue
		}

		// Log setup values
		logger.LogSetup(domainConfig.Server)

		// config server http & https
		srvHTTP := CreateServerConfig(
			domain,
			domainConfig.Server.Port.HTTP,
			domainConfig.Server.Port.HTTPS,
			nil,
		)

		srvHTTPS := CreateServerConfig(
			domain,
			domainConfig.Server.Port.HTTPS,
			domainConfig.Server.Port.HTTPS,
			&srvtls.CertificatePair{
				Cert: domainConfig.Server.TLS.CertFile,
				Key:  domainConfig.Server.TLS.KeyFile,
			},
		)

		// start server http & https
		if _, ok := serversHTTP[domainConfig.Server.Port.HTTP]; !ok {
			serversHTTP[domainConfig.Server.Port.HTTP] = srvHTTP
			go func() { log.Fatal(srvHTTP.ListenAndServe()) }()
		}
		if _, ok := serversHTTPS[domainConfig.Server.Port.HTTPS]; !ok {
			serversHTTPS[domainConfig.Server.Port.HTTPS] = srvHTTPS
			go func() { log.Fatal(srvHTTPS.ListenAndServeTLS("", "")) }()
		}

		// lb
		// TODO: HOW TO HANDLE THIS?
		balancer.InitRoundRobin(domainConfig.Server.Forwarding.Endpoints)
	}

	// Wait for an interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// Attempt a graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, v := range serversHTTP {
		v.Shutdown(ctx)
	}

	for _, v := range serversHTTPS {
		v.Shutdown(ctx)
	}
}
