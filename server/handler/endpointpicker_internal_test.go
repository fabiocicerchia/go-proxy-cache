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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/router"
)

type stubPicker struct {
	endpoint string
	err      error
	declined bool
	called   bool
}

func (s *stubPicker) Pick(context.Context, string, *http.Request, []string) (string, error) {
	s.called = true

	return s.endpoint, s.err
}

func (s *stubPicker) Declined(error) bool { return s.declined }

func pickerRoute(backend router.Backend) RequestCall {
	req := httptest.NewRequest(http.MethodPost, "http://llm.local/v1/completions", nil)
	req.Host = "llm.local"

	rc := NewRequestCall(httptest.NewRecorder(), req)
	rc.Route = &router.Route{ID: "llm", Host: "llm.local", Backends: []router.Backend{backend}}

	return rc
}

func TestPickEndpointUsesThePickersChoice(t *testing.T) {
	picker := &stubPicker{endpoint: "10.2.0.7:8000"}

	SetEndpointPicker(picker)
	defer SetEndpointPicker(nil)

	backend := router.Backend{
		Endpoints:      []string{"10.2.0.1:8000", "10.2.0.7:8000"},
		EndpointPicker: "epp.default.svc:9002",
	}

	endpoint, err := pickerRoute(backend).pickEndpoint(backend, 0)

	assert.Nil(t, err)
	assert.Equal(t, "10.2.0.7:8000", endpoint)
	assert.True(t, picker.called)
}

// A backend with no picker is balanced as before; the picker is never asked.
func TestPickEndpointSkipsThePickerWhenTheBackendHasNone(t *testing.T) {
	picker := &stubPicker{endpoint: "10.2.0.7:8000"}

	SetEndpointPicker(picker)
	defer SetEndpointPicker(nil)

	backend := router.Backend{Endpoints: []string{"10.2.0.1:8000"}}

	endpoint, err := pickerRoute(backend).pickEndpoint(backend, 0)

	assert.Nil(t, err)
	assert.Equal(t, "10.2.0.1:8000", endpoint)
	assert.False(t, picker.called, "an ordinary backend must not consult a picker")
}

// The default. A picker exists because the endpoints are not interchangeable,
// so an unreachable one means the request cannot be placed.
func TestPickEndpointFailsClosedByDefault(t *testing.T) {
	SetEndpointPicker(&stubPicker{err: errors.New("connection refused")})
	defer SetEndpointPicker(nil)

	backend := router.Backend{
		Endpoints:      []string{"10.2.0.1:8000"},
		EndpointPicker: "epp.default.svc:9002",
	}

	_, err := pickerRoute(backend).pickEndpoint(backend, 0)

	assert.Equal(t, errNoBackend, err)
}

func TestPickEndpointFailsOpenWhenAsked(t *testing.T) {
	SetEndpointPicker(&stubPicker{err: errors.New("connection refused")})
	defer SetEndpointPicker(nil)

	backend := router.Backend{
		Endpoints:              []string{"10.2.0.1:8000"},
		EndpointPicker:         "epp.default.svc:9002",
		EndpointPickerFailOpen: true,
	}

	endpoint, err := pickerRoute(backend).pickEndpoint(backend, 0)

	assert.Nil(t, err)
	assert.Equal(t, "10.2.0.1:8000", endpoint, "falls back to ordinary balancing")
}

// Failing open covers an outage, not a refusal: a picker that answered "no
// suitable endpoint" has done its job, and serving anyway would send the
// request to somewhere it just ruled out.
func TestPickEndpointDoesNotFailOpenPastADeclinedPick(t *testing.T) {
	SetEndpointPicker(&stubPicker{err: errors.New("no endpoint"), declined: true})
	defer SetEndpointPicker(nil)

	backend := router.Backend{
		Endpoints:              []string{"10.2.0.1:8000"},
		EndpointPicker:         "epp.default.svc:9002",
		EndpointPickerFailOpen: true,
	}

	_, err := pickerRoute(backend).pickEndpoint(backend, 0)

	assert.Equal(t, errNoBackend, err)
}
