package router

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
	"regexp"
	"sort"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// Table - An immutable, pre-sorted routing table.
//
// Hosts are bucketed so a lookup never scans routes belonging to a different
// virtual host, and each bucket is sorted once at build time by the matching
// precedence rules, so Match returns the first rule that matches.
type Table struct {
	exact    map[string][]*Route
	wildcard []*Route
	catchAll []*Route
	routes   []*Route
}

// Build - Compiles a routing table from a set of routes.
//
// Routes with an invalid regular expression are dropped rather than failing
// the whole table: one malformed Ingress must not take the ingress controller
// down for every other tenant in the cluster.
func Build(routes []*Route) *Table {
	t := &Table{exact: make(map[string][]*Route)}

	for _, r := range routes {
		if !compile(r) {
			continue
		}

		t.routes = append(t.routes, r)

		switch {
		case r.Host == "":
			t.catchAll = append(t.catchAll, r)
		case strings.HasPrefix(r.Host, "*."):
			t.wildcard = append(t.wildcard, r)
		default:
			host := strings.ToLower(r.Host)
			t.exact[host] = append(t.exact[host], r)
		}
	}

	for host := range t.exact {
		sortRoutes(t.exact[host])
	}

	sortRoutes(t.wildcard)
	sortRoutes(t.catchAll)

	return t
}

// Routes - Every route the table serves.
func (t *Table) Routes() []*Route {
	if t == nil {
		return nil
	}

	return t.routes
}

// Len - How many routes the table serves.
func (t *Table) Len() int {
	if t == nil {
		return 0
	}

	return len(t.routes)
}

// Match - Finds the route serving a request, following the Gateway API
// hostname precedence: an exact hostname beats a wildcard, which beats a rule
// with no hostname at all.
func (t *Table) Match(req *http.Request) (*Route, bool) {
	if t == nil {
		return nil, false
	}

	host := strings.ToLower(utils.StripPort(req.Host))

	if r, ok := matchIn(t.exact[host], req); ok {
		return r, true
	}

	if r, ok := matchWildcard(t.wildcard, host, req); ok {
		return r, true
	}

	return matchIn(t.catchAll, req)
}

func matchWildcard(routes []*Route, host string, req *http.Request) (*Route, bool) {
	for _, r := range routes {
		if !wildcardMatches(r.Host, host) {
			continue
		}

		if matches(r, req) {
			return r, true
		}
	}

	return nil, false
}

// wildcardMatches - "*.example.com" matches "a.example.com" but neither
// "example.com" nor "a.b.example.com": the wildcard covers exactly one label,
// as required by the Gateway API and by Ingress.
func wildcardMatches(pattern string, host string) bool {
	suffix := pattern[1:] // ".example.com"

	if !strings.HasSuffix(host, suffix) {
		return false
	}

	label := host[:len(host)-len(suffix)]

	return label != "" && !strings.Contains(label, ".")
}

func matchIn(routes []*Route, req *http.Request) (*Route, bool) {
	for _, r := range routes {
		if matches(r, req) {
			return r, true
		}
	}

	return nil, false
}

func matches(r *Route, req *http.Request) bool {
	return matchesPath(r, req.URL.Path) &&
		matchesMethod(r, req.Method) &&
		matchesHeaders(r, req) &&
		matchesQuery(r, req)
}

func matchesPath(r *Route, path string) bool {
	if path == "" {
		path = "/"
	}

	switch r.PathType {
	case PathExact:
		return path == r.Path
	case PathRegex:
		return r.pathRe != nil && r.pathRe.MatchString(path)
	default:
		return prefixMatches(r.Path, path)
	}
}

// prefixMatches - Whole-element prefix comparison: "/abc" matches "/abc" and
// "/abc/def" but not "/abcd". A bare "/" matches everything.
func prefixMatches(prefix string, path string) bool {
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return true
	}

	if !strings.HasPrefix(path, prefix) {
		return false
	}

	rest := path[len(prefix):]

	return rest == "" || strings.HasPrefix(rest, "/")
}

func matchesMethod(r *Route, method string) bool {
	if len(r.Methods) == 0 {
		return true
	}

	for _, m := range r.Methods {
		if strings.EqualFold(m, method) {
			return true
		}
	}

	return false
}

func matchesHeaders(r *Route, req *http.Request) bool {
	for i := range r.Headers {
		if !valueMatches(&r.Headers[i], req.Header.Get(r.Headers[i].Name)) {
			return false
		}
	}

	return true
}

func matchesQuery(r *Route, req *http.Request) bool {
	if len(r.Query) == 0 {
		return true
	}

	query := req.URL.Query()

	for i := range r.Query {
		if !valueMatches(&r.Query[i], query.Get(r.Query[i].Name)) {
			return false
		}
	}

	return true
}

func valueMatches(m *Match, actual string) bool {
	if m.Type == MatchRegex {
		return m.re != nil && m.re.MatchString(actual)
	}

	return actual == m.Value
}

func compile(r *Route) bool {
	if r.Path == "" {
		r.Path = "/"
		r.PathType = PathPrefix
	}

	if r.PathType == PathRegex {
		re, err := regexp.Compile(r.Path)
		if err != nil {
			return false
		}

		r.pathRe = re
	}

	return compileMatches(r.Headers) && compileMatches(r.Query)
}

func compileMatches(list []Match) bool {
	for i := range list {
		if list[i].Type != MatchRegex {
			continue
		}

		re, err := regexp.Compile(list[i].Value)
		if err != nil {
			return false
		}

		list[i].re = re
	}

	return true
}

// sortRoutes - Orders routes by the Gateway API matching precedence, which is
// also a superset of the Ingress "longest path wins" rule:
//
//	exact path > longest prefix > regex, then most header matches, then a
//	method match, then most query matches, then the oldest object.
func sortRoutes(routes []*Route) {
	sort.SliceStable(routes, func(i, j int) bool {
		a, b := routes[i], routes[j]

		if a.PathType != b.PathType {
			return pathRank(a.PathType) < pathRank(b.PathType)
		}

		if len(a.Path) != len(b.Path) {
			return len(a.Path) > len(b.Path)
		}

		if len(a.Headers) != len(b.Headers) {
			return len(a.Headers) > len(b.Headers)
		}

		if (len(a.Methods) > 0) != (len(b.Methods) > 0) {
			return len(a.Methods) > 0
		}

		if len(a.Query) != len(b.Query) {
			return len(a.Query) > len(b.Query)
		}

		if !a.CreationTimestamp.Equal(b.CreationTimestamp) {
			return a.CreationTimestamp.Before(b.CreationTimestamp)
		}

		return a.ID < b.ID
	})
}

func pathRank(t PathType) int {
	switch t {
	case PathExact:
		return 0
	case PathPrefix:
		return 1
	default:
		return 2
	}
}
