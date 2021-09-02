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
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// ConvertToRequestCallDTO - Generates a storage DTO containing request, response and cache settings.
func ConvertToRequestCallDTO(rc RequestCall) storage.RequestCallDTO {
	responseHeaders := http.Header{}
	if rc.Response != nil {
		responseHeaders = rc.Response.Header()
	}

	return storage.RequestCallDTO{
		Response: *rc.Response,
		Request:  rc.Request,
		CacheObject: cache.Object{
			AllowedStatuses: rc.DomainConfig.Cache.AllowedStatuses,
			AllowedMethods:  rc.DomainConfig.Cache.AllowedMethods,
			DomainID:        rc.DomainConfig.Server.Upstream.GetDomainID(),
			CurrentURIObject: cache.URIObj{
				URL:             rc.GetRequestURL(),
				Method:          rc.Request.Method,
				StatusCode:      rc.Response.StatusCode,
				RequestHeaders:  rc.Request.Header,
				ResponseHeaders: responseHeaders,
				Content:         rc.Response.Content,
			},
		},
	}
}

func getListeningPort(ctx context.Context) string {
	listeningPort := ""

	localAddrContextKey := ctx.Value(http.LocalAddrContextKey)
	if localAddrContextKey != nil {
		srvAddr := localAddrContextKey.(*net.TCPAddr)
		listeningPort = strconv.Itoa(srvAddr.Port)
	}

	return listeningPort
}

func isLegitPort(port config.Port, listeningPort string) bool {
	// When running the functional tests there's no server listening (so no port open).
	if os.Getenv("TESTING") == "1" && listeningPort == "" {
		log.Warn("Testing Environment found, and listening port is empty")
		return true
	}

	return port.HTTP == listeningPort || port.HTTPS == listeningPort
}

func (rc RequestCall) patchProxyTransport() *http.Transport {
	// G402 (CWE-295): TLS InsecureSkipVerify may be true. (Confidence: LOW, Severity: HIGH)
	// It can be ignored as it is customisable, but the default is false.
	return &http.Transport{
		MaxIdleConns:        DefaultTransportMaxIdleConns,
		MaxIdleConnsPerHost: DefaultTransportMaxIdleConnsPerHost,
		MaxConnsPerHost:     DefaultTransportMaxConnsPerHost,
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, DefaultTransportDialTimeout)
		},
		DisableKeepAlives: false,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: rc.DomainConfig.Server.Upstream.InsecureBridge,
		},
	} // #nosec
}

func getOverridePort(host string, port string, scheme string) string {
	// if there's already a port it must have priority
	if strings.Contains(host, ":") {
		return ""
	}

	portOverride := port

	if portOverride == "" && scheme == "http" {
		portOverride = "80"
	} else if portOverride == "" && scheme == "https" {
		portOverride = "443"
	}

	if portOverride != "" {
		portOverride = ":" + portOverride
	}

	return portOverride
}

// GetUpstreamURL - Get the URL based on the upstream.
func (rc RequestCall) GetUpstreamURL() url.URL {
	upstream := rc.DomainConfig.Server.Upstream
	overridePort := getOverridePort(upstream.Host, upstream.Port, rc.GetScheme())

	// Override Hostname with Destination Hostname.
	hostname := upstream.Host + overridePort
	scheme := upstream.Scheme
	if scheme == config.SchemeWildcard {
		scheme = rc.GetScheme()
	}

	lbID := upstream.Host + utils.StringSeparatorOne + upstream.Scheme
	balancedHost := balancer.GetLBRoundRobin(lbID, hostname)
	overridePort = getOverridePort(balancedHost, upstream.Port, scheme)

	return url.URL{
		Scheme: scheme,
		Host:   balancedHost + overridePort,
	}
}

// ProxyDirector - Add extra behaviour to request.
func (rc RequestCall) ProxyDirector(req *http.Request) {
	upstream := rc.DomainConfig.Server.Upstream
	overridePort := getOverridePort(upstream.Host, upstream.Port, rc.GetScheme())
	host := utils.IfEmpty(upstream.Host, upstream.Host+overridePort)

	// The value of r.URL.Host and r.Host are almost always different. On a
	// proxy server, r.URL.Host is the host of the target server and r.Host is
	// the host of the proxy server itself.
	// Ref: https://stackoverflow.com/a/42926149/888162
	req.Header.Set("X-Forwarded-Host", rc.Request.Header.Get("Host"))

	req.Header.Set("X-Forwarded-Proto", rc.GetScheme())

	previousXForwardedFor := rc.Request.Header.Get("X-Forwarded-For")
	clientIP := utils.StripPort(rc.Request.RemoteAddr)

	xForwardedFor := net.ParseIP(clientIP).String()
	if previousXForwardedFor != "" {
		xForwardedFor = previousXForwardedFor + ", " + xForwardedFor
	}

	req.Header.Set("X-Forwarded-For", xForwardedFor)

	req.Host = host
}
