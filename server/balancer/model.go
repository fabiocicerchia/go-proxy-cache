package balancer

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
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

// Balancer - Represents a Load Balancer interface.
type Balancer interface {
	GetHealthyNodes() []Item
	Pick(requestURL string) (string, error)
}
