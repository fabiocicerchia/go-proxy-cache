//go:build k8s
// +build k8s

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
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/jwt"
	"github.com/fabiocicerchia/go-proxy-cache/server/router"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	circuitbreaker "github.com/fabiocicerchia/go-proxy-cache/utils/circuit-breaker"
)

// WithK8s - Derives the served domains from the cluster's Ingress and Gateway
// API objects instead of from the configuration file.
//
// Only compiled into the build that carries the Kubernetes client, which is
// why nothing in the tag-free half of this package names a k8s type.
func WithK8s(opts k8s.Options) Option {
	return func(o *options) {
		o.start = func(s *Servers) (controller, error) {
			return s.startIngressController(opts)
		}
	}
}

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

	// One host can be served by several objects with different settings, so
	// authentication has to follow the matched route rather than the Host
	// header.
	handler.SetRouteAuthorizer(jwt.Validate)

	globalConfig := config.Config
	domainID := globalConfig.Server.Upstream.GetDomainID()

	// One Redis connection and one circuit breaker for the whole controller:
	// the cache keys already carry the host, so partitioning the connection
	// per virtual host would only multiply connections to the same server.
	logger.LogSetup(globalConfig.Server)
	circuitbreaker.InitCircuitBreaker(domainID, globalConfig.CircuitBreaker, logger.GetGlobal())
	engine.InitConn(domainID, globalConfig.Cache, logger.GetGlobal())

	// Routed mode serves whatever hosts the cluster asks it to, so the
	// per-request series have no bound anyone has agreed to. Off unless the
	// configuration says otherwise; an explicit setting is applied by Run.
	if globalConfig.Metrics.PerRequestSeries == nil {
		metrics.SetDetailedRequestSeries(false)
		logger.GetGlobal().Info("Per-request Prometheus series disabled (set metrics.per_request_series to record them)")
	}

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
