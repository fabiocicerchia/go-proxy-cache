package handler

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
)

// HandleHealthcheck - Returns healthcheck status.
func HandleHealthcheck(cfg config.Configuration) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		tracingSpan, ctx := tracing.StartSpanFromRequest("server.handle_healthcheck", req)
		defer tracingSpan.End()

		rc := NewRequestCall(res, req)
		rc.DomainConfig, _ = config.DomainConf(req.Host, rc.GetScheme())

		lwr := response.NewLoggedResponseWriter(res, rc.ReqID)

		statusCode := http.StatusOK

		domainID := config.Config.Server.Upstream.GetDomainID()
		conn := engine.GetConn(domainID)
		redisOK := conn != nil && conn.Ping()
		if !redisOK {
			logger.GetGlobal().Errorf("Redis main connection is not ok")
			statusCode = http.StatusInternalServerError
		}

		for domain, conf := range config.Config.Domains {
			domainID := conf.Server.Upstream.GetDomainID()
			conn := engine.GetConn(domainID)
			redisOK = conn != nil && conn.Ping()
			if !redisOK {
				logger.GetGlobal().Errorf("Redis connection for %s is not ok", domain)
				statusCode = http.StatusInternalServerError
			}
		}

		lwr.WriteHeader(statusCode)
		_ = lwr.WriteBody("HTTP OK\n")

		telemetry.From(ctx).RegisterStatusCode(statusCode)

		if redisOK {
			metrics.SetUp(1)
			_ = lwr.WriteBody("REDIS OK\n")
		} else {
			metrics.SetUp(0)
			_ = lwr.WriteBody("REDIS KO\n")
		}
	}
}
