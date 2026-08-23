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
	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/k8s"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

// startIngressController - Sets up the listeners and the controller for
// Kubernetes ingress mode.
//
// Unlike the static configuration, which starts one server per configured
// domain and ends up with whichever domain was processed last owning each
// port, ingress mode serves every virtual host from a single HTTP listener and
// a single HTTPS listener. The routing table decides which route serves a
// request, and the certificate store answers SNI, so there is nothing
// per-domain left to conflict over.
func (s *Servers) startIngressController(opts k8s.Options) (*k8s.Controller, error) {
	router.Enable()

	globalConfig := config.Config
	domainID := globalConfig.Server.Upstream.GetDomainID()

	// One Redis connection and one circuit breaker for the whole controller:
	// the cache keys already carry the host, so partitioning the connection
	// per virtual host would only multiply connections to the same server.
	logger.LogSetup(globalConfig.Server)
	circuitbreaker.InitCircuitBreaker(domainID, globalConfig.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, globalConfig.Cache, logger.GetGlobal())

	certs := srvtls.NewStore()
	srvtls.UseStore(certs)

	controller, err := k8s.New(opts, certs, NewRegistry())
	if err != nil {
		return nil, err
	}

	s.AttachPlain("*", globalConfig.Server.Port.HTTP, InitServer("*", globalConfig))

	srvHTTPS := InitServer("*", globalConfig)
	srvHTTPS.TLSConfig = srvtls.DynamicTLSConfig()
	s.AttachSecure("*", globalConfig.Server.Port.HTTPS, srvHTTPS)

	return controller, nil
}
