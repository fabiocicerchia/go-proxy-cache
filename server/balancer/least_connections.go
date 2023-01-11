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

// LeastConnectionsBalancer instance.
type LeastConnectionsBalancer struct {
	NodeBalancer

	connections map[string]int64
}

// NewLeastConnectionsBalancer - Creates a new instance.
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

	electedNode := healthyNodes[0].Endpoint

	b.NodeBalancer.M.RLock()
	leastConnection := b.connections[electedNode]

	for _, v := range healthyNodes {
		if b.connections[v.Endpoint] < leastConnection {
			leastConnection = b.connections[v.Endpoint]
			electedNode = v.Endpoint
		}
	}
	b.NodeBalancer.M.RUnlock()

	b.NodeBalancer.M.Lock()
	b.connections[electedNode]++
	b.NodeBalancer.M.Unlock()

	return electedNode, nil
}

// ResetCounter - Resets internal connection counters periodically.
func (b *LeastConnectionsBalancer) ResetCounter(period time.Duration) {
	go func() {
		t := time.NewTicker(period)

		for {
			<-t.C

			b.connections = make(map[string]int64)
		}
	}()
}
