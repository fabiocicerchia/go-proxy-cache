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
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	log "github.com/sirupsen/logrus"
)

type RequestCall struct {
	Response *response.LoggedResponseWriter
	Request  *http.Request
}

func getListeningPort(ctx context.Context) string {
	localAddrContextKey := ctx.Value(http.LocalAddrContextKey)
	listeningPort := ""
	if localAddrContextKey != nil {
		srvAddr := localAddrContextKey.(*net.TCPAddr)
		listeningPort = strconv.Itoa(srvAddr.Port)
	}

	return listeningPort
}

// HandleRequest - Handles the entrypoint and directs the traffic to the right handler.
func HandleRequest(res http.ResponseWriter, req *http.Request) {
	rc := RequestCall{
		Response: response.NewLoggedResponseWriter(res),
		Request:  req,
	}

	listeningPort := getListeningPort(req.Context())

	domainConfig := config.DomainConf(req.Host)
	if domainConfig == nil ||
		(domainConfig.Server.Port.HTTP != listeningPort &&
			domainConfig.Server.Port.HTTPS != listeningPort) {
		rc.Response.WriteHeader(http.StatusNotImplemented)
		logger.LogRequest(*rc.Request, *rc.Response, false)
		log.Errorf("Missing configuration in HandleRequest for %s (listening on :%s).", rc.Request.Host, listeningPort)
		return
	}

	if rc.GetScheme() == "http" && domainConfig.Server.Forwarding.HTTP2HTTPS {
		rc.RedirectToHTTPS(domainConfig.Server.Forwarding.RedirectStatusCode)
		return
	}

	if rc.Request.Method == "PURGE" {
		rc.HandlePurge(domainConfig)
		return
	}

	if req.Method == http.MethodConnect {
		rc.Response.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		rc.HandleRequestAndProxy(domainConfig)
	}
}

// GetScheme - Returns current request scheme
// For server requests the URL is parsed from the URI supplied on the
// Request-Line as stored in RequestURI. For most requests, fields other than
// Path and RawQuery will be empty. (See RFC 7230, Section 5.3)
// Ref: https://github.com/golang/go/issues/28940
func (rc RequestCall) GetScheme() string {
	if rc.Request.TLS != nil {
		// TODO: COVERAGE
		return "https"
	}
	return "http"
}
