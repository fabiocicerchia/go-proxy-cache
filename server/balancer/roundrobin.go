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

// New - Creates a new instance.
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
	b.NodeBalancer.M.RLock()
	healthyNodes := b.NodeBalancer.GetHealthyNodes()
	b.NodeBalancer.M.RUnlock()

	if len(healthyNodes) == 0 {
		return "", ErrNoAvailableItem
	}

	b.NodeBalancer.M.Lock()
	r := healthyNodes[b.next]
	b.next = (b.next + 1) % len(healthyNodes)
	b.NodeBalancer.M.Unlock()

	return r.Endpoint, nil
}
