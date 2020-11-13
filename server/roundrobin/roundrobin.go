package roundrobin

import (
	"errors"
	"sync"
)

var (
	//ErrNoAvailableItem no item is available
	ErrNoAvailableItem = errors.New("no item is available")
)

// Balancer roundrobin instance
type Balancer struct {
	m sync.Mutex

	next  int
	items []interface{}
}

// New balancer instance
func New(items []interface{}) *Balancer {
	return &Balancer{items: items}
}

// Pick available item
func (b *Balancer) Pick() (interface{}, error) {
	if len(b.items) == 0 {
		return nil, ErrNoAvailableItem
	}

	b.m.Lock()
	r := b.items[b.next]
	b.next = (b.next + 1) % len(b.items)
	b.m.Unlock()

	return r, nil
}
