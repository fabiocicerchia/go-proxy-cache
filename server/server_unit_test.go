//go:build all || unit
// +build all unit

package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newServers() *Servers {
	return &Servers{HTTP: make(map[string]*Server), HTTPS: make(map[string]*Server)}
}

func TestAttachInternalsBindsConfiguredAddress(t *testing.T) {
	s := newServers()
	public := &http.Server{}
	s.AttachPlain("example.com", "50080", public)

	internal := &http.Server{}
	err := s.AttachInternals("127.0.0.1", "52021", internal)

	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1:52021", internal.Addr)
	assert.Equal(t, ":50080", public.Addr)
	assert.Same(t, public, s.HTTP["50080"].HttpSrv)
}

func TestAttachInternalsIPv6(t *testing.T) {
	internal := &http.Server{}
	assert.NoError(t, newServers().AttachInternals("::1", "52021", internal))
	assert.Equal(t, "[::1]:52021", internal.Addr)
}

func TestAttachInternalsRejectsPublicPortCollision(t *testing.T) {
	s := newServers()
	public := &http.Server{}
	s.AttachPlain("example.com", "50080", public)
	s.AttachSecure("example.com", "50443", &http.Server{})

	assert.Error(t, s.AttachInternals("127.0.0.1", "50080", &http.Server{}))
	assert.Error(t, s.AttachInternals("0.0.0.0", "50443", &http.Server{}))
	assert.Same(t, public, s.HTTP["50080"].HttpSrv)
}
