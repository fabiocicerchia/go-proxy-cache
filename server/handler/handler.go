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

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	log "github.com/sirupsen/logrus"
)

// HandleRequest - Handles the entrypoint and directs the traffic to the right handler.
func HandleRequest(cfg config.Configuration) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		rc := initRequestParams(res, req, cfg)
		if rc.DomainConfig == nil {
			return
		}

		if rc.GetScheme() == SchemeHTTP && rc.DomainConfig.Server.Upstream.HTTP2HTTPS {
			rc.RedirectToHTTPS()
			return
		}

		if rc.Request.Method == "PURGE" {
			rc.HandlePurge()
			return
		}

		if rc.Request.Method == http.MethodConnect {
			rc.Response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if rc.IsWebSocket() {
			rc.HandleWSRequestAndProxy()
		} else {
			rc.HandleHTTPRequestAndProxy()
		}
	}
}

func initRequestParams(res http.ResponseWriter, req *http.Request, cfg config.Configuration) RequestCall {
	rc := RequestCall{
		Response: response.NewLoggedResponseWriter(res),
		Request:  req,
	}

	listeningPort := getListeningPort(req.Context())

	rc.DomainConfig = cfg.DomainConf(rc.GetHostname(), rc.GetScheme())
	if rc.DomainConfig == nil || !isLegitPort(rc.DomainConfig.Server.Port, listeningPort) {
		rc.Response.WriteHeader(http.StatusNotImplemented)
		logger.LogRequest(*rc.Request, *rc.Response, false, CacheStatusLabel[CacheStatusMiss])
		log.Errorf("Missing configuration in HandleRequest for %s (listening on :%s).", rc.Request.Host, listeningPort)

		return RequestCall{}
	}

	return rc
}
