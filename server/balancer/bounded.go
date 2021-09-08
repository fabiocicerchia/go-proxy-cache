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
)

// BoundedBalancer instance.
type BoundedBalancer struct {
	NodeBalancer

	next int
}

// New - Creates a new instance.
func NewBoundedBalancer(name string, items []Item) *BoundedBalancer {
	return &BoundedBalancer{
		NodeBalancer: NodeBalancer{
			Id:    name,
			M:     sync.Mutex{},
			Items: items,
		},
		next: 0,
	}
}

// GetHealthyNodes - Retrieves healthy nodes.
func (b BoundedBalancer) GetHealthyNodes() []Item {
	healthyNodes := []Item{}

	for _, v := range b.NodeBalancer.Items {
		if v.Healthy {
			healthyNodes = append(healthyNodes, v)
		}
	}

	return healthyNodes
}

// Pick - Chooses next available item.
func (b *BoundedBalancer) Pick() (string, error) {
	healthyNodes := b.GetHealthyNodes()

	if len(healthyNodes) == 0 {
		return "", ErrNoAvailableItem
	}

	b.NodeBalancer.M.Lock()
	r := healthyNodes[b.next]
	b.next = (b.next + 1) % len(healthyNodes)
	b.NodeBalancer.M.Unlock()

	return r.Endpoint, nil
}
