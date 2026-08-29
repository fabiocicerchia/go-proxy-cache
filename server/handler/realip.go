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
	"net"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// ForwardedProtoHeader - Carries the scheme the client originally used.
const ForwardedProtoHeader = "X-Forwarded-Proto"

// ForwardedForHeader - Carries the chain of client and proxy addresses.
const ForwardedForHeader = "X-Forwarded-For"

// isFromTrustedProxy - Whether the peer is a proxy allowed to speak for the
// client.
//
// Both X-Forwarded-Proto and X-Forwarded-For are trivially forgeable by
// anything that can reach the listener, so they are only believed when the
// connection itself comes from a configured proxy. With no trusted proxies
// configured nothing is believed, which is the right default for a proxy
// facing the internet directly.
func (rc RequestCall) isFromTrustedProxy() bool {
	trusted := rc.DomainConfig.Server.TrustedProxies
	if len(trusted) == 0 {
		return false
	}

	peer := net.ParseIP(utils.StripPort(rc.Request.RemoteAddr))
	if peer == nil {
		return false
	}

	return isIPAllowed(peer, trusted)
}

// forwardedProto - The scheme the client originally used, when a trusted proxy
// reported one.
func (rc RequestCall) forwardedProto() (string, bool) {
	if !rc.isFromTrustedProxy() {
		return "", false
	}

	value := rc.Request.Header.Get(ForwardedProtoHeader)
	if value == "" {
		return "", false
	}

	// A chain of proxies appends, so the client's own scheme is the first hop.
	proto, _, _ := strings.Cut(value, ",")

	return strings.ToLower(strings.TrimSpace(proto)), true
}

// ClientIP - The address of the actual client.
//
// Behind a trusted proxy this is the right-most X-Forwarded-For entry that is
// not itself a trusted proxy: walking from the right skips the hops we control
// and stops at the first address a client could have forged, which is the
// closest thing to the real client that can be relied on. Everything to the
// left of it is attacker-controlled and must not be used for access control.
func (rc RequestCall) ClientIP() string {
	peer := utils.StripPort(rc.Request.RemoteAddr)

	if !rc.isFromTrustedProxy() {
		return peer
	}

	trusted := rc.DomainConfig.Server.TrustedProxies
	hops := strings.Split(rc.Request.Header.Get(ForwardedForHeader), ",")

	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		if hop == "" {
			continue
		}

		ip := net.ParseIP(utils.StripPort(hop))
		if ip == nil {
			// A malformed entry means the chain cannot be trusted any
			// further; stop rather than skipping past it.
			return peer
		}

		if !isIPAllowed(ip, trusted) {
			return ip.String()
		}
	}

	return peer
}
