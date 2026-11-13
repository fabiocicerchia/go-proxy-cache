//go:build all || unit
// +build all unit

package router_test

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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

func route(id string, host string, path string, pathType router.PathType) *router.Route {
	return &router.Route{
		ID:       id,
		Host:     host,
		Path:     path,
		PathType: pathType,
		Backends: []router.Backend{{Endpoints: []string{"10.0.0.1:8080"}, Scheme: "http", Weight: 1}},
	}
}

func request(method string, url string, host string) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	req.Host = host

	return req
}

func TestMatchExactHostBeatsWildcardAndCatchAll(t *testing.T) {
	table := router.Build([]*router.Route{
		route("catchall", "", "/", router.PathPrefix),
		route("wildcard", "*.example.com", "/", router.PathPrefix),
		route("exact", "a.example.com", "/", router.PathPrefix),
	})

	r, found := table.Match(request("GET", "/", "a.example.com"))

	assert.True(t, found)
	assert.Equal(t, "exact", r.ID)
}

func TestMatchWildcardCoversExactlyOneLabel(t *testing.T) {
	table := router.Build([]*router.Route{
		route("wildcard", "*.example.com", "/", router.PathPrefix),
	})

	_, found := table.Match(request("GET", "/", "a.example.com"))
	assert.True(t, found)

	_, found = table.Match(request("GET", "/", "example.com"))
	assert.False(t, found, "the wildcard must not match the bare domain")

	_, found = table.Match(request("GET", "/", "a.b.example.com"))
	assert.False(t, found, "the wildcard must cover only one label")
}

func TestMatchIgnoresPortAndCase(t *testing.T) {
	table := router.Build([]*router.Route{
		route("exact", "a.example.com", "/", router.PathPrefix),
	})

	r, found := table.Match(request("GET", "/", "A.Example.com:8443"))

	assert.True(t, found)
	assert.Equal(t, "exact", r.ID)
}

func TestPrefixMatchesWholeElementsOnly(t *testing.T) {
	table := router.Build([]*router.Route{
		route("api", "a.example.com", "/abc", router.PathPrefix),
	})

	for _, path := range []string{"/abc", "/abc/", "/abc/def"} {
		_, found := table.Match(request("GET", path, "a.example.com"))
		assert.True(t, found, path)
	}

	_, found := table.Match(request("GET", "/abcd", "a.example.com"))
	assert.False(t, found, "/abcd must not match the /abc prefix")
}

func TestExactPathBeatsLongerPrefix(t *testing.T) {
	table := router.Build([]*router.Route{
		route("prefix", "a.example.com", "/api/v1/users", router.PathPrefix),
		route("exact", "a.example.com", "/api", router.PathExact),
	})

	r, found := table.Match(request("GET", "/api", "a.example.com"))

	assert.True(t, found)
	assert.Equal(t, "exact", r.ID)
}

func TestLongestPrefixWins(t *testing.T) {
	table := router.Build([]*router.Route{
		route("short", "a.example.com", "/", router.PathPrefix),
		route("long", "a.example.com", "/api/v1", router.PathPrefix),
		route("medium", "a.example.com", "/api", router.PathPrefix),
	})

	r, found := table.Match(request("GET", "/api/v1/users", "a.example.com"))

	assert.True(t, found)
	assert.Equal(t, "long", r.ID)
}

func TestPrefixBeatsRegex(t *testing.T) {
	table := router.Build([]*router.Route{
		route("regex", "a.example.com", "^/api/.*$", router.PathRegex),
		route("prefix", "a.example.com", "/api", router.PathPrefix),
	})

	r, found := table.Match(request("GET", "/api/x", "a.example.com"))

	assert.True(t, found)
	assert.Equal(t, "prefix", r.ID)
}

func TestRegexPathMatches(t *testing.T) {
	table := router.Build([]*router.Route{
		route("regex", "a.example.com", "^/assets/.*\\.js$", router.PathRegex),
	})

	_, found := table.Match(request("GET", "/assets/app.js", "a.example.com"))
	assert.True(t, found)

	_, found = table.Match(request("GET", "/assets/app.css", "a.example.com"))
	assert.False(t, found)
}

func TestInvalidRegexRouteIsDroppedNotFatal(t *testing.T) {
	good := route("good", "a.example.com", "/", router.PathPrefix)
	bad := route("bad", "a.example.com", "^[", router.PathRegex)

	table := router.Build([]*router.Route{bad, good})

	assert.Equal(t, 1, table.Len(), "the malformed route must be dropped")

	r, found := table.Match(request("GET", "/", "a.example.com"))
	assert.True(t, found)
	assert.Equal(t, "good", r.ID)
}

func TestMethodMatch(t *testing.T) {
	post := route("post", "a.example.com", "/", router.PathPrefix)
	post.Methods = []string{"POST"}

	table := router.Build([]*router.Route{
		route("any", "a.example.com", "/", router.PathPrefix),
		post,
	})

	r, found := table.Match(request("POST", "/", "a.example.com"))
	assert.True(t, found)
	assert.Equal(t, "post", r.ID, "a method match is more specific than none")

	r, found = table.Match(request("GET", "/", "a.example.com"))
	assert.True(t, found)
	assert.Equal(t, "any", r.ID)
}

func TestHeaderMatchCountBreaksTies(t *testing.T) {
	one := route("one", "a.example.com", "/", router.PathPrefix)
	one.Headers = []router.Match{{Name: "X-Canary", Value: "yes"}}

	two := route("two", "a.example.com", "/", router.PathPrefix)
	two.Headers = []router.Match{
		{Name: "X-Canary", Value: "yes"},
		{Name: "X-Tier", Value: "gold"},
	}

	table := router.Build([]*router.Route{one, two})

	req := request("GET", "/", "a.example.com")
	req.Header.Set("X-Canary", "yes")
	req.Header.Set("X-Tier", "gold")

	r, found := table.Match(req)
	assert.True(t, found)
	assert.Equal(t, "two", r.ID)

	req = request("GET", "/", "a.example.com")
	req.Header.Set("X-Canary", "yes")

	r, found = table.Match(req)
	assert.True(t, found)
	assert.Equal(t, "one", r.ID)
}

func TestQueryMatch(t *testing.T) {
	r1 := route("beta", "a.example.com", "/", router.PathPrefix)
	r1.Query = []router.Match{{Name: "v", Value: "beta"}}

	table := router.Build([]*router.Route{
		route("stable", "a.example.com", "/", router.PathPrefix),
		r1,
	})

	r, found := table.Match(request("GET", "/?v=beta", "a.example.com"))
	assert.True(t, found)
	assert.Equal(t, "beta", r.ID)

	r, found = table.Match(request("GET", "/?v=stable", "a.example.com"))
	assert.True(t, found)
	assert.Equal(t, "stable", r.ID)
}

func TestRegexHeaderMatch(t *testing.T) {
	r1 := route("canary", "a.example.com", "/", router.PathPrefix)
	r1.Headers = []router.Match{{Name: "X-Version", Value: "^v2\\.", Type: router.MatchRegex}}

	table := router.Build([]*router.Route{r1})

	req := request("GET", "/", "a.example.com")
	req.Header.Set("X-Version", "v2.1.0")

	_, found := table.Match(req)
	assert.True(t, found)

	req = request("GET", "/", "a.example.com")
	req.Header.Set("X-Version", "v1.9.0")

	_, found = table.Match(req)
	assert.False(t, found)
}

func TestOldestObjectWinsFullTie(t *testing.T) {
	older := route("b-older", "a.example.com", "/", router.PathPrefix)
	older.CreationTimestamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	newer := route("a-newer", "a.example.com", "/", router.PathPrefix)
	newer.CreationTimestamp = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	table := router.Build([]*router.Route{newer, older})

	r, found := table.Match(request("GET", "/", "a.example.com"))
	assert.True(t, found)
	assert.Equal(t, "b-older", r.ID)
}

func TestNoMatchReturnsFalse(t *testing.T) {
	table := router.Build([]*router.Route{
		route("api", "a.example.com", "/api", router.PathPrefix),
	})

	_, found := table.Match(request("GET", "/other", "a.example.com"))
	assert.False(t, found)

	_, found = table.Match(request("GET", "/api", "b.example.com"))
	assert.False(t, found)
}

func TestNilTableMatchesNothing(t *testing.T) {
	var table *router.Table

	_, found := table.Match(request("GET", "/", "a.example.com"))

	assert.False(t, found)
	assert.Equal(t, 0, table.Len())
}

func TestEmptyPathDefaultsToRootPrefix(t *testing.T) {
	table := router.Build([]*router.Route{route("root", "a.example.com", "", router.PathExact)})

	_, found := table.Match(request("GET", "/anything", "a.example.com"))

	assert.True(t, found, "an empty path must behave as a / prefix")
}
