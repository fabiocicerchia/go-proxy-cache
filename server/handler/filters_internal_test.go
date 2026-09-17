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
	"net/http"
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
