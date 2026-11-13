//go:build all || unit
// +build all unit

package epp_test

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
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	extprocv3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"

	"github.com/fabiocicerchia/go-proxy-cache/server/epp"
)

// fakePicker - An ext-proc server standing in for a real Endpoint Picker.
//
// The wire format is the only thing worth testing here and it cannot be
// checked against a running picker in CI, so this speaks the protocol
// literally: it records what the proxy sent and answers the way the proposal
// says a picker answers.
type fakePicker struct {
	extprocv3.UnimplementedExternalProcessorServer

	respond func(*extprocv3.ProcessingRequest) *extprocv3.ProcessingResponse

	gotSubset    string
	gotPath      string
	gotAuthority string
}

func (f *fakePicker) Process(stream extprocv3.ExternalProcessor_ProcessServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	subset := req.GetMetadataContext().GetFilterMetadata()[epp.SubsetHintNamespace]
	f.gotSubset = subset.GetFields()[epp.SubsetHintKey].GetStringValue()

	for _, h := range req.GetRequestHeaders().GetHeaders().GetHeaders() {
		switch h.GetKey() {
		case ":path":
			f.gotPath = string(h.GetRawValue())
		case ":authority":
			f.gotAuthority = string(h.GetRawValue())
		}
	}

	return stream.Send(f.respond(req))
}

func startPicker(t *testing.T, picker *fakePicker) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot listen: %s", err)
	}

	srv := grpc.NewServer()
	extprocv3.RegisterExternalProcessorServer(srv, picker)

	go func() { _ = srv.Serve(listener) }()

	t.Cleanup(srv.Stop)

	return listener.Addr().String()
}

func headerResponse(value string) *extprocv3.ProcessingResponse {
	return &extprocv3.ProcessingResponse{
		Response: &extprocv3.ProcessingResponse_RequestHeaders{
			RequestHeaders: &extprocv3.HeadersResponse{
				Response: &extprocv3.CommonResponse{
					HeaderMutation: &extprocv3.HeaderMutation{
						SetHeaders: []*corev3.HeaderValueOption{{
							Header: &corev3.HeaderValue{
								Key:      epp.DestinationHeader,
								RawValue: []byte(value),
							},
						}},
					},
				},
			},
		},
	}
}

func TestPickReadsTheEndpointFromTheHeader(t *testing.T) {
	picker := &fakePicker{
		respond: func(*extprocv3.ProcessingRequest) *extprocv3.ProcessingResponse {
			return headerResponse("10.2.0.7:8000")
		},
	}

	client := epp.NewClient(2 * time.Second)
	defer client.Close()

	req := httptest.NewRequest("POST", "http://llm.local/v1/chat/completions", nil)
	req.Host = "llm.local"

	endpoint, err := client.Pick(context.Background(), startPicker(t, picker),
		req, []string{"10.2.0.1:8000", "10.2.0.7:8000"})

	assert.Nil(t, err)
	assert.Equal(t, "10.2.0.7:8000", endpoint)

	// The candidate set has to reach the picker, or it is choosing blind.
	assert.Equal(t, "10.2.0.1:8000,10.2.0.7:8000", picker.gotSubset)
	assert.Equal(t, "/v1/chat/completions", picker.gotPath)
	assert.Equal(t, "llm.local", picker.gotAuthority)
}

// The protocol allows the answer in dynamic metadata instead of a header.
func TestPickReadsTheEndpointFromDynamicMetadata(t *testing.T) {
	picker := &fakePicker{
		respond: func(*extprocv3.ProcessingRequest) *extprocv3.ProcessingResponse {
			return &extprocv3.ProcessingResponse{
				Response: &extprocv3.ProcessingResponse_RequestHeaders{
					RequestHeaders: &extprocv3.HeadersResponse{},
				},
				DynamicMetadata: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						epp.DestinationMetadataNamespace: structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								epp.DestinationHeader: structpb.NewStringValue("10.2.0.9:8000"),
							},
						}),
					},
				},
			}
		},
	}

	client := epp.NewClient(2 * time.Second)
	defer client.Close()

	endpoint, err := client.Pick(context.Background(), startPicker(t, picker),
		httptest.NewRequest("POST", "http://llm.local/v1/completions", nil), []string{"10.2.0.9:8000"})

	assert.Nil(t, err)
	assert.Equal(t, "10.2.0.9:8000", endpoint)
}

// A list is retry candidates in order; the first is the choice.
func TestPickTakesTheFirstOfSeveralEndpoints(t *testing.T) {
	picker := &fakePicker{
		respond: func(*extprocv3.ProcessingRequest) *extprocv3.ProcessingResponse {
			return headerResponse("10.2.0.3:8000,10.2.0.4:8000")
		},
	}

	client := epp.NewClient(2 * time.Second)
	defer client.Close()

	endpoint, err := client.Pick(context.Background(), startPicker(t, picker),
		httptest.NewRequest("POST", "http://llm.local/v1/completions", nil),
		[]string{"10.2.0.3:8000", "10.2.0.4:8000"})

	assert.Nil(t, err)
	assert.Equal(t, "10.2.0.3:8000", endpoint)
}

// "No suitable endpoint" is an immediate 503, and has to be distinguishable
// from the picker being unreachable: one is a decision, the other is an
// outage, and the failure modes treat them differently.
func TestPickTreatsAnImmediateResponseAsDeclined(t *testing.T) {
	picker := &fakePicker{
		respond: func(*extprocv3.ProcessingRequest) *extprocv3.ProcessingResponse {
			return &extprocv3.ProcessingResponse{
				Response: &extprocv3.ProcessingResponse_ImmediateResponse{
					ImmediateResponse: &extprocv3.ImmediateResponse{
						Status: &typev3.HttpStatus{Code: typev3.StatusCode_ServiceUnavailable},
					},
				},
			}
		},
	}

	client := epp.NewClient(2 * time.Second)
	defer client.Close()

	_, err := client.Pick(context.Background(), startPicker(t, picker),
		httptest.NewRequest("POST", "http://llm.local/v1/completions", nil), []string{"10.2.0.1:8000"})

	assert.NotNil(t, err)
	assert.True(t, client.Declined(err), "a declined pick is not the same as an unreachable picker")
}

func TestPickTreatsAnUnreachablePickerAsAnOutage(t *testing.T) {
	client := epp.NewClient(300 * time.Millisecond)
	defer client.Close()

	// Nothing listening: the port is closed immediately after being taken.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)

	address := listener.Addr().String()
	_ = listener.Close()

	_, err = client.Pick(context.Background(), address,
		httptest.NewRequest("POST", "http://llm.local/v1/completions", nil), []string{"10.2.0.1:8000"})

	assert.NotNil(t, err)
	assert.False(t, client.Declined(err), "an unreachable picker is an outage, not a decision")
}

func TestPickWithoutCandidatesDeclines(t *testing.T) {
	client := epp.NewClient(time.Second)
	defer client.Close()

	_, err := client.Pick(context.Background(), "127.0.0.1:1",
		httptest.NewRequest("POST", "http://llm.local/v1/completions", nil), nil)

	assert.True(t, client.Declined(err))
}
