package tls

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	crypto_tls "crypto/tls"
	"strings"
	"sync"
)

// Store - A hot-swappable set of certificates indexed by the hostnames they
// serve.
//
// The file-based certificates map in this package is written once at boot and
// read from the SNI callback without synchronisation, which is safe only
// because nothing ever changes it. Certificates sourced from Kubernetes
// Secrets do change while traffic is flowing, so they live here instead,
// behind an RWMutex, and with the wildcard matching that the exact-map lookup
// never had.
type Store struct {
	mu       sync.RWMutex
	exact    map[string]*crypto_tls.Certificate
	wildcard map[string]*crypto_tls.Certificate
	fallback *crypto_tls.Certificate
}

// NewStore - An empty certificate store.
func NewStore() *Store {
	return &Store{
		exact:    make(map[string]*crypto_tls.Certificate),
		wildcard: make(map[string]*crypto_tls.Certificate),
	}
}

// Replace - Swaps the whole certificate set.
//
// Keys are the hostnames each certificate serves; a "*.example.com" key is
// indexed as a wildcard. A "" key, if present, becomes the fallback served
// when SNI matches nothing (and when the client sends no SNI at all).
func (s *Store) Replace(certs map[string]*crypto_tls.Certificate) {
	exact := make(map[string]*crypto_tls.Certificate, len(certs))
	wildcard := make(map[string]*crypto_tls.Certificate)

	var fallback *crypto_tls.Certificate

	for host, cert := range certs {
		switch {
		case host == "":
			fallback = cert
		case strings.HasPrefix(host, "*."):
			wildcard[strings.ToLower(host[2:])] = cert
		default:
			exact[strings.ToLower(host)] = cert
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.exact = exact
	s.wildcard = wildcard
	s.fallback = fallback
}

// Get - The certificate serving a hostname.
//
// An exact match wins over a wildcard, which covers exactly one label:
// "*.example.com" serves "a.example.com" but neither "example.com" nor
// "a.b.example.com".
func (s *Store) Get(serverName string) (*crypto_tls.Certificate, bool) {
	if s == nil {
		return nil, false
	}

	host := strings.ToLower(strings.TrimSuffix(serverName, "."))

	s.mu.RLock()
	defer s.mu.RUnlock()

	if cert, ok := s.exact[host]; ok {
		return cert, true
	}

	if _, rest, found := strings.Cut(host, "."); found && rest != "" {
		if cert, ok := s.wildcard[rest]; ok {
			return cert, true
		}
	}

	if s.fallback != nil {
		return s.fallback, true
	}

	return nil, false
}

// Len - How many hostnames the store serves.
func (s *Store) Len() int {
	if s == nil {
		return 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.exact) + len(s.wildcard)
}

// dynamicStore - The store consulted by the SNI callback, set when the proxy
// runs as an ingress controller.
var dynamicStore *Store

// UseStore - Registers the store the SNI callback falls back to.
func UseStore(s *Store) {
	dynamicStore = s
}

// DynamicTLSConfig - A TLS configuration that resolves every certificate
// through the store at handshake time, for the single HTTPS listener the
// ingress controller runs.
func DynamicTLSConfig() *crypto_tls.Config {
	return newDefaultTLSConfig()
}
