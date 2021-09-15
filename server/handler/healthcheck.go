package handler

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/tracing"
	"github.com/opentracing/opentracing-go"
)

// HandleHealthcheck - Returns healthcheck status.
func HandleHealthcheck(cfg config.Configuration) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		tracingSpan := tracing.StartSpanFromRequest("server.handle_healthcheck", req)
		defer tracingSpan.Finish()
		ctx := opentracing.ContextWithSpan(req.Context(), tracingSpan)

		rc, err := initRequestParams(ctx, res, req)
		if err != nil {
			tracing.AddErrorToSpan(tracingSpan, err)
			tracing.Fail(tracingSpan, "internal error")

			rc.GetLogger().Errorf(err.Error())
			return
		}

		domainID := rc.DomainConfig.Server.Upstream.GetDomainID()

		lwr := response.NewLoggedResponseWriter(res, rc.ReqID)

		statusCode := http.StatusOK

		conn := engine.GetConn(domainID)
		redisOK := conn != nil && conn.Ping()
		if !redisOK {
			statusCode = http.StatusInternalServerError
		}

		lwr.WriteHeader(statusCode)
		_ = lwr.WriteBody("HTTP OK\n")

		tracingSpan.SetTag("response.status_code", statusCode)

		if redisOK {
			_ = lwr.WriteBody("REDIS OK\n")
		} else {
			_ = lwr.WriteBody("REDIS KO\n")
		}
	}
}
