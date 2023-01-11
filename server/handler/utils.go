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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/dnscache"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/tracing"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// RequestIDHeader - HTTP Header to be forwarded to the upstream backend.
const RequestIDHeader = "X-Go-Proxy-Cache-Request-ID"

var r *dnscache.Resolver = &dnscache.Resolver{}

// ConvertToRequestCallDTO - Generates a storage DTO containing request, response and cache settings.
func ConvertToRequestCallDTO(rc RequestCall) storage.RequestCallDTO {
	responseHeaders := http.Header{}
	if rc.Response != nil {
		responseHeaders = rc.Response.Header()
	}

	return storage.RequestCallDTO{
		ReqID:    rc.ReqID,
		Response: *rc.Response,
		Request:  rc.Request,
		CacheObject: cache.Object{
			ReqID:           rc.ReqID,
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
		logger.GetGlobal().Warn("Testing Environment found, and listening port is empty")
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
		DialContext: func(ctx context.Context, network string, address string) (conn net.Conn, err error) {
			// DNS Cache
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := r.LookupHost(ctx, host)
			if err != nil {
				return nil, err
			}

			// Timeout Dial
			d := net.Dialer{Timeout: DefaultTransportDialTimeout}

			for _, ip := range ips {
				conn, err = d.DialContext(ctx, network, net.JoinHostPort(ip, port))
				if err == nil {
					return conn, err
				}
			}

			return d.DialContext(ctx, network, address)
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
func (rc RequestCall) GetUpstreamURL() (url.URL, error) {
	upstream := rc.DomainConfig.Server.Upstream
	overridePort := getOverridePort(upstream.Host, upstream.Port, rc.GetScheme())

	// Override Hostname with Destination Hostname.
	hostname := upstream.Host + overridePort

	balancedEndpoint := balancer.GetUpstreamNode(upstream.GetDomainID(), rc.GetRequestURL(), hostname)
	if !strings.Contains(balancedEndpoint, "://") {
		// Ref: https://github.com/golang/go/issues/19297#issuecomment-282651469
		balancedEndpoint = fmt.Sprintf("//%s", balancedEndpoint)
	}
	balancedURL, err := url.Parse(balancedEndpoint)
	if err != nil {
		return url.URL{}, err
	}

	// scheme
	scheme := upstream.Scheme
	if scheme == config.SchemeWildcard {
		scheme = rc.GetScheme()
	}
	// use scheme only when full scheme + domain (+ port) is provided as endpoint.
	if balancedURL.Scheme != "" && balancedURL.Host != "" {
		scheme = balancedURL.Scheme
	}

	// host
	balancedHost := balancedURL.Host
	// when it's specified only the hostname, url.Parse it converts it to Path.
	if balancedHost == "" {
		balancedHost = balancedEndpoint
	}
	if balancedHost != "" && balancedHost != upstream.Host {
		hostname = balancedHost
	}

	// port
	upstreamPort := upstream.Port
	_, port, _ := net.SplitHostPort(hostname)
	// if port is defined in endpoint, it takes the precedence over listening port.
	if port != "" && port != upstreamPort {
		upstreamPort = port
	}

	overridePort = getOverridePort(hostname, upstreamPort, scheme)

	return url.URL{
		Scheme: scheme,
		User:   balancedURL.User,
		Host:   hostname + overridePort,
	}, nil
}

// GetUpstreamHost - Retrieve the real upstream host
func (rc RequestCall) GetUpstreamHost() string {
	upstream := rc.DomainConfig.Server.Upstream
	overridePort := getOverridePort(upstream.Host, upstream.Port, rc.GetScheme())
	host := utils.IfEmpty(upstream.Host, upstream.Host+overridePort)

	return host
}

// ProxyDirector - Add extra behaviour to request.
func (rc RequestCall) ProxyDirector(span opentracing.Span) func(req *http.Request) {
	return func(req *http.Request) {
		upstreamHost := rc.GetUpstreamHost()

		// The value of r.URL.Host and r.Host are almost always different. On a
		// proxy server, r.URL.Host is the host of the target server and r.Host is
		// the host of the proxy server itself.
		// Ref: https://stackoverflow.com/a/42926149/888162
		req.Header.Set("X-Forwarded-Host", rc.Request.Header.Get("Host"))

		req.Header.Set("X-Forwarded-Proto", rc.GetScheme())

		req.Header.Set(RequestIDHeader, rc.ReqID)

		previousXForwardedFor := rc.Request.Header.Get("X-Forwarded-For")
		clientIP := utils.StripPort(rc.Request.RemoteAddr)

		xForwardedFor := net.ParseIP(clientIP).String()
		if previousXForwardedFor != "" {
			xForwardedFor = previousXForwardedFor + ", " + xForwardedFor
		}

		req.Header.Set("X-Forwarded-For", xForwardedFor)

		req.Host = upstreamHost

		_ = tracing.Inject(span, req)
	}
}
