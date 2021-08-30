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
)

// HandleHealthcheck - Returns healthcheck status.
func HandleHealthcheck(cfg config.Configuration) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		rc := initRequestParams(res, req, cfg)
		domainID := rc.DomainConfig.Server.Upstream.GetDomainID()

		lwr := response.NewLoggedResponseWriter(res)

		lwr.WriteHeader(http.StatusOK)
		_ = lwr.WriteBody("HTTP OK\n")

		if conn := engine.GetConn(domainID); conn != nil && conn.Ping() {
			_ = lwr.WriteBody("REDIS OK\n")
		} else {
			_ = lwr.WriteBody("REDIS KO\n")
		}
	}
}
