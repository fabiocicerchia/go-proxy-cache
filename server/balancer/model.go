package balancer

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"sync"
	"time"
)

// HealthCheckInterval - Health Check Frequency.
const HealthCheckInterval time.Duration = 30 * time.Second // TODO: make it customizable

// LeastConnectionsResetInterval - How often reset internal counter for Least Connection LoadBalancer.
const LeastConnectionsResetInterval time.Duration = 5 * time.Minute // TODO: make it customizable

// LoadBalancing - Contains the multiple instances for the active servers.
type LoadBalancing map[string]Balancer

var lb LoadBalancing

// lbMu - Guards the lb map.
//
// The map used to be written once at boot and read lock-free from the request
// path. The Kubernetes ingress controller adds and removes balancers while
// traffic is flowing, so every access is now synchronised.
var lbMu sync.RWMutex

// stopHealthChecks - Per-balancer stop channels for the health-check
// goroutines, so a balancer that goes away does not leak its ticker.
var stopHealthChecks = make(map[string]chan struct{})

// Item - Represents a load balanced node.
type Item struct {
	Healthy  bool
	Endpoint string
}

// NodeBalancer - Core structure for a load balancer.
type NodeBalancer struct {
	M sync.RWMutex

	Id    string
	Items []Item
}

// GetNodeBalancer - Returns the embedded NodeBalancer, promoted to every
// concrete balancer type so Init* can health-check without a type assertion.
func (b *NodeBalancer) GetNodeBalancer() *NodeBalancer {
	return b
}

// Balancer - Represents a Load Balancer interface.
type Balancer interface {
	GetHealthyNodes() []Item
	Pick(requestURL string) (string, error)
	GetNodeBalancer() *NodeBalancer
}
