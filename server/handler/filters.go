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
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/server/router"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
)

// applyRequestFilters - Applies a route's request-side filters to the
// outgoing upstream request: header mutations and URL rewriting.
//
// Runs inside the reverse proxy director, after the default director has set
// the scheme, host and path, so a rewritten path is the one actually sent.
func applyRequestFilters(route *router.Route, req *http.Request) {
	for i := range route.Filters {
		f := &route.Filters[i]

		switch f.Type {
		case router.FilterRequestHeaderModifier:
			applyHeaderOp(req.Header, f.RequestHeaders)
		case router.FilterURLRewrite:
			applyRewrite(route, f.Rewrite, req)
		}
	}
}

// applyResponseFilters - Applies a route's response-side header mutations.
func applyResponseFilters(route *router.Route, header http.Header) {
	for i := range route.Filters {
		f := &route.Filters[i]

		if f.Type == router.FilterResponseHeaderModifier {
			applyHeaderOp(header, f.ResponseHeaders)
		}
	}
}

func applyHeaderOp(header http.Header, op *router.HeaderOp) {
	if op == nil {
		return
	}

	for k, v := range op.Set {
		header.Set(k, v)
	}

	for k, v := range op.Add {
		header.Add(k, v)
	}

	for _, k := range op.Remove {
		header.Del(k)
	}
}

func applyRewrite(route *router.Route, rewrite *router.Rewrite, req *http.Request) {
	if rewrite == nil {
		return
	}

	if rewrite.Hostname != "" {
		req.Host = rewrite.Hostname
	}

	if rewrite.ReplaceFullPath != nil {
		req.URL.Path = *rewrite.ReplaceFullPath
		return
	}

	if rewrite.ReplacePrefixMatch != nil {
		req.URL.Path = replacePrefix(route.Path, *rewrite.ReplacePrefixMatch, req.URL.Path)
	}
}

// replacePrefix - Swaps the portion of the path matched by the route's prefix
// for a replacement, keeping the remainder intact. "/api" -> "/" applied to
// "/api/v1/x" yields "/v1/x".
func replacePrefix(prefix string, replacement string, path string) string {
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return path
	}

	if !strings.HasPrefix(path, prefix) {
		return path
	}

	rest := strings.TrimPrefix(path[len(prefix):], "/")
	replacement = strings.TrimSuffix(replacement, "/")

	if rest == "" {
		if replacement == "" {
			return "/"
		}

		return replacement
	}

	return replacement + "/" + rest
}

// redirectFilter - Returns the route's redirect filter, if it has one.
func redirectFilter(route *router.Route) *router.Redirect {
	for i := range route.Filters {
		if route.Filters[i].Type == router.FilterRequestRedirect {
			return route.Filters[i].Redirect
		}
	}

	return nil
}

// HandleRouteRedirect - Answers a request with the redirect its route defines,
// instead of proxying it upstream.
func (rc RequestCall) HandleRouteRedirect(ctx context.Context, redirect *router.Redirect) {
	scheme := redirect.Scheme
	if scheme == "" {
		scheme = rc.GetScheme()
	}

	host := redirect.Hostname
	if host == "" {
		host = rc.GetHostname()
	}

	if redirect.Port > 0 {
		host = fmt.Sprintf("%s:%s", host, strconv.Itoa(redirect.Port))
	}

	path := rc.Request.URL.Path
	if redirect.Path != nil {
		switch {
		case redirect.Path.ReplaceFullPath != nil:
			path = *redirect.Path.ReplaceFullPath
		case redirect.Path.ReplacePrefixMatch != nil:
			path = replacePrefix(rc.Route.Path, *redirect.Path.ReplacePrefixMatch, path)
		}
	}

	target := scheme + "://" + host + path
	if rc.Request.URL.RawQuery != "" {
		target += "?" + rc.Request.URL.RawQuery
	}

	statusCode := redirect.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusFound
	}

	rc.Response.Header().Set("Location", target)
	rc.Response.ForceWriteHeader(statusCode)

	telemetry.From(ctx).RegisterStatusCode(statusCode)
}
