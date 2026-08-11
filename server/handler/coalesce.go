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
	"io"
	"net/http"

	"golang.org/x/sync/singleflight"
)

// reverseProxyGroup - Coalesces concurrent identical upstream fetches
// (collapsed forwarding), so a cache-miss stampede for the same resource
// hits the origin once instead of once per concurrent request.
var reverseProxyGroup singleflight.Group

// coalescedResponse - Buffered upstream response shared across all callers
// waiting on the same singleflight key.
type coalescedResponse struct {
	resp *http.Response
	body []byte
}

// coalescingRoundTripper - http.RoundTripper decorator that collapses
// concurrent GET/HEAD requests for the same URL into a single upstream
// round-trip. Only GET/HEAD are coalesced: they're safe to share across
// callers, unlike POST/PURGE which carry per-caller semantics/bodies.
//
// ponytail: keyed on method+URL only, not Vary-aware (matches how the
// upstream response's Vary headers aren't known until after the fetch).
// Upgrade to a Vary-aware key if a coalesced backend actually varies
// per-request in a way that matters.
type coalescingRoundTripper struct {
	http.RoundTripper
}

// proxyTransport - Returns the transport for the reverse proxy, wrapped in the
// coalescing decorator only when collapsed forwarding is enabled for the domain
// (feature flag, off by default).
func (rc RequestCall) proxyTransport() http.RoundTripper {
	if rc.DomainConfig.Server.Upstream.CollapsedForwarding {
		return coalescingRoundTripper{rc.patchProxyTransport()}
	}

	return rc.patchProxyTransport()
}

func (c coalescingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return c.RoundTripper.RoundTrip(req)
	}

	key := req.Method + " " + req.URL.String()

	v, err, _ := reverseProxyGroup.Do(key, func() (interface{}, error) {
		resp, err := c.RoundTripper.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}

		return &coalescedResponse{resp: resp, body: body}, nil
	})
	if err != nil {
		return nil, err
	}

	cr := v.(*coalescedResponse)

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
