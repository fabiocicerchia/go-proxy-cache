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
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/fabiocicerchia/go-proxy-cache/utils/random"
)

// IpHashBalancer instance.
type IpHashBalancer struct {
	NodeBalancer

	hashMap map[string]int64
}

// NewIpHashBalancer - Creates a new instance.
func NewIpHashBalancer(name string, items []Item) *IpHashBalancer {
	return &IpHashBalancer{
		NodeBalancer: NodeBalancer{
			Id:    name,
			M:     sync.RWMutex{},
			Items: items,
		},
		hashMap: make(map[string]int64),
	}
}

// GetHealthyNodes - Retrieves healthy nodes.
func (b *IpHashBalancer) GetHealthyNodes() []Item {
	healthyNodes := []Item{}

	for _, v := range b.NodeBalancer.Items {
		if v.Healthy {
			healthyNodes = append(healthyNodes, v)
		}
	}

	return healthyNodes
}

// Pick - Chooses next available item.
func (b *IpHashBalancer) Pick(requestURL string) (string, error) {
	healthyNodes := b.NodeBalancer.GetHealthyNodes()
	if len(healthyNodes) == 0 {
		return "", ErrNoAvailableItem
	}

	h := sha256.New()
	h.Write([]byte(requestURL))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	b.NodeBalancer.M.RLock()
	if pos, ok := b.hashMap[hash]; ok {
		b.NodeBalancer.M.RUnlock()
		return healthyNodes[pos].Endpoint, nil
	}
	b.NodeBalancer.M.RUnlock()

	rnd := random.RandomInt64(int64(len(healthyNodes)))

	r := healthyNodes[rnd]
	b.NodeBalancer.M.Lock()
	b.hashMap[hash] = rnd
	b.NodeBalancer.M.Unlock()

	return r.Endpoint, nil
}
