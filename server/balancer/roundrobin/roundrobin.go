//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache
package roundrobin

import (
	"errors"
	"sync"
)

var (
	// ErrNoAvailableItem no item is available
	ErrNoAvailableItem = errors.New("no item is available")
)

// Balancer roundrobin instance.
type Balancer struct {
	m sync.Mutex

	next  int
	items []string
}

// New - Creates a new instance.
func New(items []string) *Balancer {
	return &Balancer{items: items}
}

// Pick - Chooses next available item.
func (b *Balancer) Pick() (string, error) {
	if len(b.items) == 0 {
		return "", ErrNoAvailableItem
	}

	b.m.Lock()
	r := b.items[b.next]
	b.next = (b.next + 1) % len(b.items)
	b.m.Unlock()

	return r, nil
}
