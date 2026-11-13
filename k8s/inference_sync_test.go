//go:build all || unit
// +build all unit

package k8s

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	inferencefake "sigs.k8s.io/gateway-api-inference-extension/client-go/clientset/versioned/fake"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"

	"github.com/fabiocicerchia/go-proxy-cache/server/router"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
)

// The Inference Extension CRDs are optional, and most clusters running Gateway
// API do not have them. Their informer can therefore never sync, and waiting on
// it would keep Run() from ever publishing a routing table -- leaving the proxy
// listening and answering 404 to everything.
func TestRunPublishesRoutesWhenInferenceCachesNeverSync(t *testing.T) {
	defer router.Reset()

	ing := simpleIngress("default", "demo", "demo.local", "/", networkingv1.PathTypePrefix, "web", 80)

	core := fake.NewSimpleClientset(
		ingressClass(IngressClassName, ControllerName, false),
		ing,
		service("default", "web", 80, "http"),
		endpointSlice("default", "web", "http", 8080, "10.1.0.1"),
	)

	// What an absent CRD looks like to the reflector: the LIST never succeeds,
	// so HasSynced never flips.
	inference := inferencefake.NewClientset()
	inference.PrependReactor("list", "inferencepools",
		func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New(`the server could not find the requested resource`)
		})

	opts := gatewayOptions()
	opts.ResyncPeriod = time.Hour
	opts.DisableStatusUpdates = true

	c := newWithClients(opts, core, gatewayfake.NewClientset(), inference, srvtls.NewStore(), &noopRegistry{})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	// The routing table must appear without waiting for the whole context.
	published := false

	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); {
		if router.Current() != nil {
			published = true

			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	assert.True(t, published, "a missing InferencePool CRD must not stop the routing table being published")

	cancel()
	<-done
}
