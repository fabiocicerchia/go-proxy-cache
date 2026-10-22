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
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/utils/ttl"
	"golang.org/x/sync/singleflight"
)

// reverseProxyGroup - Coalesces concurrent identical upstream fetches
// (collapsed forwarding), so a cache-miss stampede for the same resource
// hits the origin once instead of once per concurrent request.
var reverseProxyGroup singleflight.Group

// coalescedResponse - Buffered upstream response shared across all callers
// waiting on the same singleflight key.
type coalescedResponse struct {
	resp  *http.Response
	body  []byte
	share bool
}

// coalescingRoundTripper - http.RoundTripper decorator that collapses
// concurrent GET/HEAD requests for the same URL into a single upstream
// round-trip.
type coalescingRoundTripper struct {
	http.RoundTripper
	scope string
}

// proxyTransport - Returns the transport for the reverse proxy, wrapped in the
// coalescing decorator only when collapsed forwarding is enabled for the domain
// (feature flag, off by default).
func (rc RequestCall) proxyTransport() http.RoundTripper {
	if rc.DomainConfig.Server.Upstream.CollapsedForwarding {
		return coalescingRoundTripper{RoundTripper: rc.patchProxyTransport(), scope: rc.DomainConfig.Server.Upstream.GetDomainID()}
	}

	return rc.patchProxyTransport()
}

func (c coalescingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return c.RoundTripper.RoundTrip(req)
	}
	uri := cache.URIObj{RequestHeaders: req.Header}
	if uri.BypassesSharedCache() {
		return c.RoundTripper.RoundTrip(req)
	}

	headers, err := json.Marshal(req.Header)
	if err != nil {
		return c.RoundTripper.RoundTrip(req)
	}
	key := fmt.Sprintf("%s %p %s %s %s %x", c.scope, c.RoundTripper, req.Method, req.URL.String(), req.Host, sha256.Sum256(headers))
	leader := false

	v, err, _ := reverseProxyGroup.Do(key, func() (interface{}, error) {
		leader = true
		resp, err := c.RoundTripper.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		uri.ResponseHeaders = resp.Header
		directives := ttl.ParseCacheControl(resp.Header)
		_, public := directives["public"]
		if !public || !uri.IsStorableBySharedCache() || resp.Header.Get("Vary") == "*" || resp.StatusCode != http.StatusOK {
			return &coalescedResponse{resp: resp}, nil
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
		if err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
		if len(body) > 1<<20 {
			resp.Body = &prefixedBody{Reader: io.MultiReader(bytes.NewReader(body), resp.Body), Closer: resp.Body}
			return &coalescedResponse{resp: resp}, nil
		}
		_ = resp.Body.Close()

		return &coalescedResponse{resp: resp, body: body, share: true}, nil
	})
	if err != nil {
		return nil, err
	}

	cr := v.(*coalescedResponse)
	if !cr.share {
		if leader {
			return cr.resp, nil
		}
		return c.RoundTripper.RoundTrip(req)
	}

	// Each waiter gets its own *http.Response/body/header, so downstream
	// per-request logic (ETag, gzip, cache storage) can freely read/mutate
	// its copy without racing the other waiters.
	respCopy := *cr.resp
	respCopy.Request = req
	respCopy.Header = cr.resp.Header.Clone()
	respCopy.Body = io.NopCloser(bytes.NewReader(cr.body))
	respCopy.ContentLength = int64(len(cr.body))

	return &respCopy, nil
}

type prefixedBody struct {
	io.Reader
	io.Closer
}
