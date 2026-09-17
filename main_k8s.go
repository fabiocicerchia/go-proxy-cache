//go:build k8s
// +build k8s

package main

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"flag"
	"os"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/k8s"
	"github.com/fabiocicerchia/go-proxy-cache/server"
)

var k8sEnabled bool
var k8sOptions k8s.Options
var publishStatusAddress string

// registerExtraFlags - Kubernetes ingress controller flags.
//
// Every flag defaults from the environment, because a Deployment configures a
// controller through env vars far more naturally than through an args list.
func registerExtraFlags() {
	defaults := k8s.NewOptions()

	flag.BoolVar(&k8sEnabled, "k8s", os.Getenv("INGRESS_CONTROLLER_ENABLED") == "true",
		"run as a Kubernetes ingress controller, deriving routes from Ingress and Gateway API objects")
	flag.BoolVar(&k8sOptions.EnableGatewayAPI, "gateway-api", os.Getenv("GATEWAY_API_ENABLED") == "true",
		"also watch GatewayClass, Gateway and HTTPRoute objects")
	flag.StringVar(&k8sOptions.IngressClass, "ingress-class", defaults.IngressClass,
		"name of the IngressClass to serve")
	flag.StringVar(&k8sOptions.ControllerName, "controller-name", defaults.ControllerName,
		"identity matched against IngressClass.spec.controller and GatewayClass.spec.controllerName")
	flag.StringVar(&k8sOptions.WatchNamespace, "watch-namespace", defaults.WatchNamespace,
		"restrict the controller to one namespace (default: whole cluster)")
	flag.StringVar(&k8sOptions.PublishService, "publish-service", defaults.PublishService,
		"namespace/name of the Service whose address is written into the status of served objects")
	flag.StringVar(&publishStatusAddress, "publish-status-address", os.Getenv("PUBLISH_STATUS_ADDRESS"),
		"comma separated addresses to publish, instead of looking them up from -publish-service")
	flag.StringVar(&k8sOptions.ElectionID, "election-id", defaults.ElectionID,
		"name of the Lease electing the replica that writes status")
	flag.StringVar(&k8sOptions.KubeConfig, "kubeconfig", os.Getenv("KUBECONFIG"),
		"path to a kubeconfig file (default: in-cluster configuration)")
	flag.BoolVar(&k8sOptions.DisableStatusUpdates, "disable-status-updates", os.Getenv("DISABLE_STATUS_UPDATES") == "true",
		"never write status back, and never run for leader election")
}

func applyExtraFlags() {
	if publishStatusAddress != "" {
		k8sOptions.PublishStatusAddress = strings.Split(publishStatusAddress, ",")
	}
}

// runOptions - Ingress mode when asked for, otherwise the config file.
func runOptions() []server.Option {
	if !k8sEnabled {
		return nil
	}

	return []server.Option{server.WithK8s(k8sOptions)}
}
