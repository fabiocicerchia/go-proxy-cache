//go:build all || unit
// +build all unit

package tls_test

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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
)

func cert(id string) *crypto_tls.Certificate {
	return &crypto_tls.Certificate{Certificate: [][]byte{[]byte(id)}}
}

func idOf(c *crypto_tls.Certificate) string {
	if c == nil || len(c.Certificate) == 0 {
		return ""
	}

	return string(c.Certificate[0])
}

func TestStoreExactMatch(t *testing.T) {
	s := srvtls.NewStore()
	s.Replace(map[string]*crypto_tls.Certificate{"a.example.com": cert("a")})

	got, ok := s.Get("a.example.com")

	assert.True(t, ok)
	assert.Equal(t, "a", idOf(got))
}

func TestStoreIsCaseInsensitiveAndIgnoresRootLabel(t *testing.T) {
	s := srvtls.NewStore()
	s.Replace(map[string]*crypto_tls.Certificate{"a.example.com": cert("a")})

	got, ok := s.Get("A.Example.COM.")

	assert.True(t, ok)
	assert.Equal(t, "a", idOf(got))
}

func TestStoreWildcard(t *testing.T) {
	s := srvtls.NewStore()
	s.Replace(map[string]*crypto_tls.Certificate{"*.example.com": cert("wild")})

	got, ok := s.Get("a.example.com")
	assert.True(t, ok)
	assert.Equal(t, "wild", idOf(got))

	// A wildcard covers exactly one label.
	_, ok = s.Get("a.b.example.com")
	assert.False(t, ok)

	_, ok = s.Get("example.com")
	assert.False(t, ok)
}

func TestStoreExactBeatsWildcard(t *testing.T) {
	s := srvtls.NewStore()
	s.Replace(map[string]*crypto_tls.Certificate{
		"*.example.com": cert("wild"),
		"a.example.com": cert("exact"),
	})

	got, ok := s.Get("a.example.com")

	assert.True(t, ok)
	assert.Equal(t, "exact", idOf(got))
}

func TestStoreFallback(t *testing.T) {
	s := srvtls.NewStore()
	s.Replace(map[string]*crypto_tls.Certificate{
		"":              cert("default"),
		"a.example.com": cert("a"),
	})

	// An unmatched SNI (or none at all) gets the fallback rather than a
	// handshake failure with no diagnostics.
	got, ok := s.Get("unknown.test")
	assert.True(t, ok)
	assert.Equal(t, "default", idOf(got))

	got, ok = s.Get("")
	assert.True(t, ok)
	assert.Equal(t, "default", idOf(got))
}

func TestStoreMissWithNoFallback(t *testing.T) {
	s := srvtls.NewStore()
	s.Replace(map[string]*crypto_tls.Certificate{"a.example.com": cert("a")})

	_, ok := s.Get("b.example.com")

	assert.False(t, ok)
}

func TestStoreReplaceDropsOldCertificates(t *testing.T) {
	s := srvtls.NewStore()
	s.Replace(map[string]*crypto_tls.Certificate{"old.example.com": cert("old")})
	s.Replace(map[string]*crypto_tls.Certificate{"new.example.com": cert("new")})

	_, ok := s.Get("old.example.com")
	assert.False(t, ok, "a Secret that was deleted must stop being served")

	_, ok = s.Get("new.example.com")
	assert.True(t, ok)
	assert.Equal(t, 1, s.Len())
}

func TestNilStoreIsSafe(t *testing.T) {
	var s *srvtls.Store

	_, ok := s.Get("a.example.com")

	assert.False(t, ok)
	assert.Equal(t, 0, s.Len())
}

func TestStoreConcurrentGetAndReplace(t *testing.T) {
	s := srvtls.NewStore()
	s.Replace(map[string]*crypto_tls.Certificate{"a.example.com": cert("a")})

	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()

			for j := 0; j < 500; j++ {
				s.Replace(map[string]*crypto_tls.Certificate{
					"a.example.com": cert("a"),
					"*.example.com": cert("wild"),
				})
			}
		}()

		go func() {
			defer wg.Done()

			for j := 0; j < 500; j++ {
				_, _ = s.Get("a.example.com")
				_, _ = s.Get("b.example.com")
			}
		}()
	}

	wg.Wait()
}
