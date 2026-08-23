package router

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import "sync/atomic"

// current - The routing table currently serving traffic.
//
// Nil until the ingress controller publishes one, which is what keeps the
// static configuration path untouched: Enabled() stays false and the handler
// never consults the router.
var current atomic.Pointer[Table]

// enabled - Whether the proxy runs in routed (ingress controller) mode.
var enabled atomic.Bool

// Enable - Switches the proxy into routed mode.
func Enable() {
	enabled.Store(true)
}

// Enabled - Whether requests should be resolved through the routing table
// rather than through the static per-domain configuration.
func Enabled() bool {
	return enabled.Load()
}

// Current - The routing table currently serving traffic, possibly nil.
func Current() *Table {
	return current.Load()
}

// Publish - Atomically swaps the routing table serving traffic.
func Publish(t *Table) {
	current.Store(t)
}

// Reset - Clears the routing table and leaves routed mode. Only meant for
// tests.
func Reset() {
	current.Store(nil)
	enabled.Store(false)
}
