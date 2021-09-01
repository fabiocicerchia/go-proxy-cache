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
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

// HandleRequest - Handles the entrypoint and directs the traffic to the right handler.
func HandleRequest(res http.ResponseWriter, req *http.Request) {
	rc, err := initRequestParams(res, req)
	if err != nil {
		log.Errorf(err.Error())
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

func initRequestParams(res http.ResponseWriter, req *http.Request) (RequestCall, error) {
	var configFound bool

	rc := RequestCall{
		Response: response.NewLoggedResponseWriter(res),
		Request:  *req,
	}

	listeningPort := getListeningPort(req.Context())

	rc.DomainConfig, configFound = config.DomainConf(req.Host, rc.GetScheme())
	if !configFound || !rc.IsLegitRequest(listeningPort) {
		rc.Response.WriteHeader(http.StatusNotImplemented)
		logger.LogRequest(rc.Request, *rc.Response, false, CacheStatusLabel[CacheStatusMiss])

		return RequestCall{}, fmt.Errorf("Request for %s (listening on :%s) is not allowed (mostly likely it's a configuration mismatch).", rc.Request.Host, listeningPort)
	}

	return rc, nil
}
