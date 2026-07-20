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
	// The set of healthy nodes can shrink between calls (e.g. a node becomes
	// unhealthy), so a previously stored position may point past the current
	// slice. Only reuse it when it is still in range, otherwise fall through and
	// pick (and store) a fresh position.
	if pos, ok := b.hashMap[hash]; ok && pos < int64(len(healthyNodes)) {
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
