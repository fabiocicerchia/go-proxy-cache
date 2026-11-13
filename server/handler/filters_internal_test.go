//go:build all || unit
// +build all unit

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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

// URL.Path is the decoded path, so an escaped separator in the original
// request is already a real one by the time a rewrite sees it. Clearing
// RawPath does not bring it back, and Go re-encodes from Path whenever the two
// disagree, so the rewritten path is what reaches the upstream either way.
func TestRewritePathReachesTheUpstreamAfterAPrefixRewrite(t *testing.T) {
	replacement := "/"

	// What the director leaves behind for an inbound /api/a%2Fb.
	req := &http.Request{URL: &url.URL{Path: "/api/a/b", RawPath: "/api/a%2Fb"}}
	route := &router.Route{Path: "/api"}

	applyRewrite(route, &router.Rewrite{ReplacePrefixMatch: &replacement}, req)

	assert.Equal(t, "/a/b", req.URL.Path)
	assert.Equal(t, "/a/b", req.URL.RequestURI(),
		"the stale RawPath no longer encodes Path, so it is ignored rather than sent")
}

func TestRewriteFullPath(t *testing.T) {
	replacement := "/v2/items"

	req := &http.Request{URL: &url.URL{Path: "/api/items", RawPath: "/api/items"}}
	route := &router.Route{Path: "/api"}

	applyRewrite(route, &router.Rewrite{ReplaceFullPath: &replacement}, req)

	assert.Equal(t, replacement, req.URL.Path)
	assert.Equal(t, replacement, req.URL.RequestURI())
}

func TestRewriteHostnameOnly(t *testing.T) {
	req := &http.Request{URL: &url.URL{Path: "/api/items"}, Host: "old.example.com"}

	applyRewrite(&router.Route{Path: "/api"}, &router.Rewrite{Hostname: "new.example.com"}, req)

	assert.Equal(t, "new.example.com", req.Host)
	assert.Equal(t, "/api/items", req.URL.Path, "a hostname-only rewrite leaves the path alone")
}

// URL.Path is the decoded path, so pasting it into a header emits a raw space
// for %20 and a real separator for %2F. Building the Location through url.URL
// escapes it.
func TestRouteRedirectEscapesTheLocation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		target   string
		expected string
	}{
		// Preserved only because the path is passed through; a rewrite
		// decodes it, which is the documented limitation of rewriting Path.
		{"encoded separator", "/a%2Fb", "http://example.com/a%2Fb"},
		{"space", "/foo%20bar", "http://example.com/foo%20bar"},
		{"plain", "/plain", "http://example.com/plain"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.target, nil)
		req.Host = "example.com"

		res := httptest.NewRecorder()
		rc := NewRequestCall(res, req)
		rc.Route = &router.Route{Path: "/"}

		rc.HandleRouteRedirect(context.Background(), &router.Redirect{StatusCode: http.StatusFound})

		assert.Equal(t, tc.expected, res.Header().Get("Location"), tc.name)
		assert.NotContains(t, res.Header().Get("Location"), " ", tc.name)
	}
}

func TestRouteRedirectKeepsTheQueryString(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?q=a%20b&n=2", nil)
	req.Host = "example.com"

	res := httptest.NewRecorder()
	rc := NewRequestCall(res, req)
	rc.Route = &router.Route{Path: "/"}

	rc.HandleRouteRedirect(context.Background(), &router.Redirect{
		Scheme:     "https",
		StatusCode: http.StatusMovedPermanently,
	})

	assert.Equal(t, "https://example.com/search?q=a%20b&n=2", res.Header().Get("Location"))
	assert.Equal(t, http.StatusMovedPermanently, res.Code)
}
