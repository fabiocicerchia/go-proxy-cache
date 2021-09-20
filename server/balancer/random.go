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
	"math/rand"
	"sync"
)

// RandomBalancer instance.
type RandomBalancer struct {
	NodeBalancer

	next int
}

// NewRandomBalancer - Creates a new instance.
func NewRandomBalancer(name string, items []Item) *RandomBalancer {
	return &RandomBalancer{
		NodeBalancer: NodeBalancer{
			Id:    name,
			M:     sync.RWMutex{},
			Items: items,
		},
		next: 0,
	}
}

// Pick - Chooses next available item.
func (b *RandomBalancer) Pick(requestURL string) (string, error) {
	healthyNodes := b.NodeBalancer.GetHealthyNodes()
	if len(healthyNodes) == 0 {
		return "", ErrNoAvailableItem
	}

	rnd := rand.Intn(len(healthyNodes))
	b.NodeBalancer.M.Lock()
	r := healthyNodes[rnd]
	b.NodeBalancer.M.Unlock()

	return r.Endpoint, nil
}
