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
	"strings"

	"github.com/yhat/wsutil"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	log "github.com/sirupsen/logrus"
)

// RequestCall - Main object containing request and response.
type RequestCall struct {
	Response *response.LoggedResponseWriter
	Request  *http.Request
}

// ConvertToRequestCallDTO - Generates a storage DTO containing request, response and cache settings.
func ConvertToRequestCallDTO(rc RequestCall) storage.RequestCallDTO {
	return storage.RequestCallDTO{
		Response: *rc.Response,
		Request:  *rc.Request,
		Scheme:   rc.GetScheme(),
		// TODO: convert to use domainConfigCache
		CacheObj: cache.CacheObj{
			AllowedStatuses: config.Config.Cache.AllowedStatuses,
			AllowedMethods:  config.Config.Cache.AllowedMethods,
		},
	}
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

	host := strings.Split(req.Host, ":")[0] // TODO: HACK
	domainConfig := config.DomainConf(host, rc.GetScheme())
	if domainConfig == nil ||
		(domainConfig.Server.Port.HTTP != listeningPort &&
			domainConfig.Server.Port.HTTPS != listeningPort) {
		rc.Response.WriteHeader(http.StatusNotImplemented)
		logger.LogRequest(*rc.Request, *rc.Response, false)
		log.Errorf("Missing configuration in HandleRequest for %s (listening on :%s).", rc.Request.Host, listeningPort)
		return
	}

	if rc.GetScheme() == "http" && domainConfig.Server.Upstream.HTTP2HTTPS {
		rc.RedirectToHTTPS(domainConfig.Server.Upstream.RedirectStatusCode)
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

// GetScheme - Returns current request scheme.
// For server requests the URL is parsed from the URI supplied on the
// Request-Line as stored in RequestURI. For most requests, fields other than
// Path and RawQuery will be empty. (See RFC 7230, Section 5.3)
// Ref: https://github.com/golang/go/issues/28940
func (rc RequestCall) GetScheme() string {
	// TODO: COVERAGE
	if rc.IsWebSocket() && rc.Request.TLS != nil {
		return "wss"
	}

	if rc.IsWebSocket() {
		return "ws"
	}

	if rc.Request.TLS != nil {
		return "https"
	}

	return "http"
}

// IsWebSocket - Checks whether a request is a websocket.
func (rc RequestCall) IsWebSocket() bool {
	return wsutil.IsWebSocketRequest(rc.Request)
}
