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
	"strconv"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// ConvertToRequestCallDTO - Generates a storage DTO containing request, response and cache settings.
func ConvertToRequestCallDTO(rc RequestCall) storage.RequestCallDTO {
	return storage.RequestCallDTO{
		Response: *rc.Response,
		Request:  *rc.Request,
		Hostname: rc.GetHostname(),
		Scheme:   rc.GetScheme(),
		CacheObject: cache.Object{
			AllowedStatuses: rc.DomainConfig.Cache.AllowedStatuses,
			AllowedMethods:  rc.DomainConfig.Cache.AllowedMethods,
			DomainID:        rc.GetHostname() + utils.StringSeparatorOne + rc.GetConfiguredScheme(),
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

func (rc *RequestCall) patchRequestForReverseProxy(upstream config.Upstream) *url.URL {
	overridePort := getOverridePort(upstream.Host, upstream.Port, rc.GetScheme())
	targetURL := *rc.Request.URL
	targetURL.Scheme = rc.GetScheme()
	targetURL.Host = upstream.Host + overridePort

	rc.FixRequest(targetURL, upstream)

	proxyURL := &url.URL{
		Scheme: rc.Request.URL.Scheme,
		Host:   rc.Request.URL.Host,
	}

	return proxyURL
}
