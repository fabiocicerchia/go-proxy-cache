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
	"errors"
	"sync"
)

// ErrNoAvailableItem no item is available.
var ErrNoAvailableItem = errors.New("no item is available")

// RoundRobinBalancer instance.
type RoundRobinBalancer struct {
	NodeBalancer

	next int
}

// NewRoundRobinBalancer - Creates a new instance.
func NewRoundRobinBalancer(name string, items []Item) *RoundRobinBalancer {
	return &RoundRobinBalancer{
		NodeBalancer: NodeBalancer{
			Id:    name,
			M:     sync.RWMutex{},
			Items: items,
		},
		next: 0,
	}
}

// Pick - Chooses next available item.
func (b *RoundRobinBalancer) Pick(requestURL string) (string, error) {
	// GetHealthyNodes locks internally.
	healthyNodes := b.NodeBalancer.GetHealthyNodes()

	if len(healthyNodes) == 0 {
		return "", ErrNoAvailableItem
	}

	b.NodeBalancer.M.Lock()
	// The set of healthy nodes can shrink between calls (e.g. a node becomes
	// unhealthy), so b.next may point past the current slice. Clamp it to avoid
	// an index-out-of-range panic.
	if b.next >= len(healthyNodes) {
		b.next = 0
	}
	r := healthyNodes[b.next]
	b.next = (b.next + 1) % len(healthyNodes)
	b.NodeBalancer.M.Unlock()

	return r.Endpoint, nil
}
