package epp

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

// Package epp talks to a Gateway API Inference Extension Endpoint Picker.
//
// The picker decides which model server should answer a request, using signals
// an ordinary load balancer cannot see -- KV-cache utilisation, queue depth,
// which adapters a pod has loaded. The protocol is Envoy's external processing
// service: the proxy opens a bidirectional stream, sends the request headers
// along with the endpoints it considers eligible, and the picker answers with
// the one to use.
//
// Ref: gateway-api-inference-extension, proposal 004 (endpoint picker protocol).

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extprocv3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// SubsetHintNamespace - Filter metadata namespace carrying the endpoints
	// the proxy is willing to use.
	SubsetHintNamespace = "envoy.lb.subset_hint"

	// SubsetHintKey - Key under SubsetHintNamespace holding them, comma
	// separated.
	SubsetHintKey = "x-gateway-destination-endpoint-subset"

	// DestinationMetadataNamespace - Dynamic metadata namespace the picker
	// answers under.
	DestinationMetadataNamespace = "envoy.lb"

	// DestinationHeader - Header the picker answers with. The same key is used
	// inside the dynamic metadata.
	DestinationHeader = "x-gateway-destination-endpoint"
)

// ErrNoEndpoint - The picker declined to choose, which it signals with an
// immediate 503.
var ErrNoEndpoint = errors.New("endpoint picker returned no endpoint")

// Client - A pool of connections to endpoint pickers, keyed by address.
//
// Connections are long-lived and shared: a picker is consulted on every
// inference request, and dialling per request would add a handshake to each
// one.
type Client struct {
	mu    sync.RWMutex
	conns map[string]*grpc.ClientConn

	// timeout - How long a single pick may take before the request gives up
	// and the failure mode decides what happens.
	timeout time.Duration
}

// NewClient - Builds a client with the given per-pick timeout.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &Client{conns: make(map[string]*grpc.ClientConn), timeout: timeout}
}

// DefaultTimeout - Per-pick budget.
//
// A picker sits in the request path, so this is deliberately short: a slow
// picker should degrade to ordinary balancing (or a 503, per the pool's
// failure mode) rather than hold the client's request open.
const DefaultTimeout = 500 * time.Millisecond

// Close - Tears down every pooled connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for addr, conn := range c.conns {
		_ = conn.Close()
		delete(c.conns, addr)
	}
}

func (c *Client) connection(address string) (*grpc.ClientConn, error) {
	c.mu.RLock()
	conn, ok := c.conns[address]
	c.mu.RUnlock()

	if ok {
		return conn, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Another request may have dialled while this one waited for the lock.
	if conn, ok := c.conns[address]; ok {
		return conn, nil
	}

	// In-cluster, pod to pod. The extension does not define a TLS story for
	// this hop, and the reference picker serves plaintext h2c.
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	c.conns[address] = conn

	return conn, nil
}

// Pick - Asks the picker which of the candidate endpoints should serve a
// request.
//
// Returns ErrNoEndpoint when the picker declines. Any other error is a
// transport or protocol failure, which the caller resolves through the pool's
// failure mode rather than here: this package does not know whether failing
// open is acceptable.
func (c *Client) Pick(ctx context.Context, address string, req *http.Request, candidates []string) (string, error) {
	if address == "" {
		return "", fmt.Errorf("no endpoint picker address")
	}

	if len(candidates) == 0 {
		return "", ErrNoEndpoint
	}

	conn, err := c.connection(address)
	if err != nil {
		return "", fmt.Errorf("cannot reach endpoint picker %s: %w", address, err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	stream, err := extprocv3.NewExternalProcessorClient(conn).Process(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot open endpoint picker stream: %w", err)
	}

	// Half-close once the headers are sent: a pick needs no more from us, and
	// leaving the stream open would keep the picker waiting for a body.
	defer func() { _ = stream.CloseSend() }()

	if err := stream.Send(processingRequest(req, candidates)); err != nil {
		return "", fmt.Errorf("cannot send to endpoint picker: %w", err)
	}

	response, err := stream.Recv()
	if err != nil {
		return "", fmt.Errorf("cannot read from endpoint picker: %w", err)
	}

	return endpointFrom(response)
}

// processingRequest - The request headers plus the eligible endpoints.
func processingRequest(req *http.Request, candidates []string) *extprocv3.ProcessingRequest {
	headers := &corev3.HeaderMap{Headers: make([]*corev3.HeaderValue, 0, len(req.Header)+3)}

	// Pseudo-headers first: the picker routes on :path and needs the authority
	// to tell virtual hosts apart.
	add := func(key string, value string) {
		headers.Headers = append(headers.Headers, &corev3.HeaderValue{
			Key:      key,
			RawValue: []byte(value),
		})
	}

	add(":method", req.Method)
	add(":path", req.URL.RequestURI())
	add(":authority", req.Host)

	for key, values := range req.Header {
		for _, value := range values {
			add(strings.ToLower(key), value)
		}
	}

	return &extprocv3.ProcessingRequest{
		Request: &extprocv3.ProcessingRequest_RequestHeaders{
			RequestHeaders: &extprocv3.HttpHeaders{
				Headers: headers,
				// No body is being forwarded, so as far as the picker is
				// concerned the request ends here.
				EndOfStream: true,
			},
		},
		MetadataContext: subsetHint(candidates),
	}
}

// subsetHint - The endpoints the proxy will accept, in the namespace the
// protocol reserves for them.
func subsetHint(candidates []string) *corev3.Metadata {
	return &corev3.Metadata{
		FilterMetadata: map[string]*structpb.Struct{
			SubsetHintNamespace: {
				Fields: map[string]*structpb.Value{
					SubsetHintKey: structpb.NewStringValue(strings.Join(candidates, ",")),
				},
			},
		},
	}
}

// endpointFrom - The chosen endpoint, from whichever of the two channels the
// picker used.
//
// The protocol allows the answer in a response header or in dynamic metadata,
// and pickers do both, so both are read. A list means retry candidates in
// order; the first is the choice.
func endpointFrom(response *extprocv3.ProcessingResponse) (string, error) {
	if response.GetImmediateResponse() != nil {
		// How the picker says "nothing suitable": an immediate 503.
		return "", ErrNoEndpoint
	}

	if endpoint := headerEndpoint(response); endpoint != "" {
		return endpoint, nil
	}

	if endpoint := metadataEndpoint(response); endpoint != "" {
		return endpoint, nil
	}

	return "", ErrNoEndpoint
}

func headerEndpoint(response *extprocv3.ProcessingResponse) string {
	mutation := response.GetRequestHeaders().GetResponse().GetHeaderMutation()

	for _, option := range mutation.GetSetHeaders() {
		header := option.GetHeader()
		if header == nil || !strings.EqualFold(header.GetKey(), DestinationHeader) {
			continue
		}

		// Newer Envoy protos carry the value in RawValue and leave the
		// deprecated string field empty.
		value := string(header.GetRawValue())
		if value == "" {
			value = header.GetValue()
		}

		if endpoint := firstEndpoint(value); endpoint != "" {
			return endpoint
		}
	}

	return ""
}

func metadataEndpoint(response *extprocv3.ProcessingResponse) string {
	metadata := response.GetDynamicMetadata()
	if metadata == nil {
		return ""
	}

	namespace := metadata.GetFields()[DestinationMetadataNamespace].GetStructValue()
	if namespace == nil {
		// Some pickers answer with the key at the top level rather than under
		// the namespace.
		return firstEndpoint(metadata.GetFields()[DestinationHeader].GetStringValue())
	}

	return firstEndpoint(namespace.GetFields()[DestinationHeader].GetStringValue())
}

// firstEndpoint - The first entry of a comma-separated endpoint list.
func firstEndpoint(value string) string {
	for _, candidate := range strings.Split(value, ",") {
		if candidate = strings.TrimSpace(candidate); candidate != "" {
			return candidate
		}
	}

	return ""
}

// Declined - Whether an error means the picker chose to serve nothing, as
// opposed to being unreachable.
//
// The distinction is what failure modes turn on: a picker that answered "no
// suitable endpoint" has done its job, and failing open past that would send
// the request somewhere it just ruled out.
func (c *Client) Declined(err error) bool {
	return errors.Is(err, ErrNoEndpoint)
}
