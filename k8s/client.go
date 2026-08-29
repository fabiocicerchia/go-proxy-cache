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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

// restConfig - Builds the API server connection, preferring the in-cluster
// service account and falling back to a kubeconfig file (which is how the
// controller is run outside a cluster during development).
func restConfig(kubeConfigPath string) (*rest.Config, error) {
	if kubeConfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	}

	cfg, err := rest.InClusterConfig()
	if err == nil {
		return cfg, nil
	}

	// Not running in a pod: fall back to the usual kubeconfig discovery
	// (KUBECONFIG, then ~/.kube/config).
	rules := clientcmd.NewDefaultClientConfigLoadingRules()

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{}).ClientConfig()
}

// newClients - Builds the core and Gateway API clientsets.
func newClients(kubeConfigPath string) (kubernetes.Interface, gatewayclientset.Interface, error) {
	cfg, err := restConfig(kubeConfigPath)
	if err != nil {
		return nil, nil, err
	}

	// A controller talks to the API server constantly during a resync storm;
	// the client-go defaults (5 QPS) throttle that into minutes of latency.
	cfg.QPS = 50
	cfg.Burst = 100

	core, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	gateway, err := gatewayclientset.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	return core, gateway, nil
}
