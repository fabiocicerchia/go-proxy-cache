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

// LeastConnectionsBalancer instance.
type LeastConnectionsBalancer struct {
	NodeBalancer

	connections map[string]int64
}

// New - Creates a new instance.
func NewLeastConnectionsBalancer(name string, items []Item) *LeastConnectionsBalancer {
	b := &LeastConnectionsBalancer{
		NodeBalancer: NodeBalancer{
			Id:    name,
			M:     sync.RWMutex{},
			Items: items,
		},
		connections: make(map[string]int64),
	}
	b.ResetCounter(LeastConnectionsResetInterval)

	return b
}

// GetHealthyNodes - Retrieves healthy nodes.
func (b *LeastConnectionsBalancer) GetHealthyNodes() []Item {
	healthyNodes := []Item{}

	for _, v := range b.NodeBalancer.Items {
		if v.Healthy {
			healthyNodes = append(healthyNodes, v)
		}
	}

	return healthyNodes
}

// Pick - Chooses next available item.
func (b *LeastConnectionsBalancer) Pick(requestURL string) (string, error) {
	healthyNodes := b.NodeBalancer.GetHealthyNodes()
	if len(healthyNodes) == 0 {
		return "", ErrNoAvailableItem
	}

	elected_node := healthyNodes[0].Endpoint

	b.NodeBalancer.M.RLock()
	least_connection := b.connections[elected_node]

	for _, v := range healthyNodes {
		if b.connections[v.Endpoint] < least_connection {
			least_connection = b.connections[v.Endpoint]
			elected_node = v.Endpoint
		}
	}
	b.NodeBalancer.M.RUnlock()

	b.NodeBalancer.M.Lock()
	b.connections[elected_node]++
	b.NodeBalancer.M.Unlock()

	return elected_node, nil
}

// CheckHealth() - Period check on nodes status.
func (b *LeastConnectionsBalancer) ResetCounter(period time.Duration) {
	go func() {
		t := time.NewTicker(period)

		for {
			<-t.C

			b.connections = make(map[string]int64)
		}
	}()
}
