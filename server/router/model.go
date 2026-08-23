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
	"regexp"
	"strconv"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// PathType - How a route's path is compared against the request path.
type PathType uint8

const (
	// PathExact - The request path must equal the route path.
	PathExact PathType = iota
	// PathPrefix - The route path must be a whole-element prefix of the request
	// path: "/abc" matches "/abc" and "/abc/def", but not "/abcd".
	PathPrefix
	// PathRegex - The route path is a regular expression matched against the
	// whole request path.
	PathRegex
)

// MatchType - How a header or query parameter value is compared.
type MatchType uint8

const (
	// MatchExact - Values must be equal.
	MatchExact MatchType = iota
	// MatchRegex - The value must match a regular expression.
	MatchRegex
)

// Match - A header or query parameter match.
type Match struct {
	Name  string
	Value string
	Type  MatchType

	re *regexp.Regexp
}

// FilterType - The kind of transformation a Filter applies.
type FilterType uint8

const (
	// FilterRequestHeaderModifier - Mutates request headers before proxying.
	FilterRequestHeaderModifier FilterType = iota
	// FilterResponseHeaderModifier - Mutates response headers before sending.
	FilterResponseHeaderModifier
	// FilterRequestRedirect - Answers the request with a redirect.
	FilterRequestRedirect
	// FilterURLRewrite - Rewrites the hostname and/or path sent upstream.
	FilterURLRewrite
)

// HeaderOp - Header mutations to apply.
type HeaderOp struct {
	Set    map[string]string
	Add    map[string]string
	Remove []string
}

// Rewrite - Hostname and path rewriting applied to the upstream request.
type Rewrite struct {
	Hostname string
	// ReplaceFullPath - Replaces the whole path when non-nil.
	ReplaceFullPath *string
	// ReplacePrefixMatch - Replaces the portion of the path matched by the
	// route's PathPrefix when non-nil.
	ReplacePrefixMatch *string
}

// Redirect - A redirect response returned instead of proxying.
type Redirect struct {
	Scheme     string
	Hostname   string
	Port       int
	StatusCode int
	Path       *Rewrite
}

// Filter - A transformation applied to a matched request or its response.
type Filter struct {
	Type            FilterType
	RequestHeaders  *HeaderOp
	ResponseHeaders *HeaderOp
	Redirect        *Redirect
	Rewrite         *Rewrite
}

// Backend - A weighted set of upstream endpoints ("10.1.2.3:8080").
type Backend struct {
	Endpoints []string
	Scheme    string
	Weight    int32
	// Name - Human readable backend identity, used in logs and as part of the
	// load balancer key.
	Name string
}

// Route - A single host+path rule resolved to its upstream backends.
type Route struct {
	// Host - Exact hostname, "*.example.com" wildcard, or "" for any host.
	Host     string
	Path     string
	PathType PathType

	Methods []string
	Headers []Match
	Query   []Match

	Backends []Backend
	Filters  []Filter

	// Config - The cache/upstream/TLS settings resolved for this route, from
	// the global configuration plus any annotations on the source object.
	Config config.Configuration

	// PreserveHost - Forward the client's Host header upstream. This is the
	// Ingress default, and the opposite of the static configuration's
	// behaviour, where the upstream host always replaces it.
	PreserveHost bool
	// UpstreamHost - Explicit Host header override, when PreserveHost is false.
	UpstreamHost string

	// ID - Stable identity, used as the load balancer key.
	ID string
	// Source - "Ingress default/demo" or "HTTPRoute default/demo", for logs
	// and status reporting.
	Source string

	CreationTimestamp time.Time

	pathRe *regexp.Regexp
}

// BalancerID - The load balancer key for one of the route's backends.
func (r *Route) BalancerID(idx int) string {
	return r.ID + utils.StringSeparatorOne + strconv.Itoa(idx)
}

// Upstream - The upstream configuration for one of the route's backends, in
// the shape the existing balancer expects.
func (r *Route) Upstream(idx int) config.Upstream {
	upstream := r.Config.Server.Upstream
	upstream.Endpoints = r.Backends[idx].Endpoints
	upstream.Scheme = r.Backends[idx].Scheme

	return upstream
}
