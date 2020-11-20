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
