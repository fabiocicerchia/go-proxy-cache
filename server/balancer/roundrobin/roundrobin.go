package roundrobin

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
	"fmt"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// TODO: make it customizable
const HealthCheckInterval time.Duration = 30 * time.Second

// ErrNoAvailableItem no item is available.
var ErrNoAvailableItem = errors.New("no item is available")

type Item struct {
	healthy  bool
	endpoint string
}

// Balancer roundrobin instance.
type Balancer struct {
	m sync.Mutex

	id     string
	next   int
	items  []Item
	logger log.Logger
}

// New - Creates a new instance.
func New(name string, items []string, enableHealthchecks bool) *Balancer {
	newItems := []Item{}
	for _, v := range items {
		item := Item{true, v}
		newItems = append(newItems, item)
	}

	b := &Balancer{
		id:     name,
		m:      sync.Mutex{},
		next:   0,
		items:  newItems,
		logger: *log.StandardLogger(),
	}

	if enableHealthchecks {
		b.CheckHealth()
	}

	return b
}

// GetHealthyNodes - Retrieves healthy nodes.
func (b Balancer) GetHealthyNodes() []Item {
	healthyNodes := []Item{}

	for _, v := range b.items {
		if v.healthy {
			healthyNodes = append(healthyNodes, v)
		}
	}

	return healthyNodes
}

// Pick - Chooses next available item.
func (b *Balancer) Pick() (string, error) {
	b.m.Lock()
	healthyNodes := b.GetHealthyNodes()
	b.m.Unlock()

	if len(healthyNodes) == 0 {
		return "", ErrNoAvailableItem
	}

	b.m.Lock()
	r := healthyNodes[b.next]
	b.next = (b.next + 1) % len(healthyNodes)
	b.m.Unlock()

	return r.endpoint, nil
}

// CheckHealth() - Period check on nodes status.
func (b *Balancer) CheckHealth() {
	period := HealthCheckInterval // todo customize

	go func() {
		t := time.NewTicker(period)

		for {
			<-t.C

			client := http.Client{
				// return the 301/302
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
				Timeout: 5 * time.Second, // TODO: make it custom
			}

			for k, v := range b.items {
				endpointURL := fmt.Sprintf("http://%s", v.endpoint) // todo fix

				req, err := http.NewRequest("GET", endpointURL, nil)
				if err != nil {
					b.logger.Errorf("Healthcheck request %s failed for %s: %s", b.id, endpointURL, err)
					continue
				}
				res, err := client.Do(req)

				v.healthy = err == nil
				if err != nil {
					b.logger.Errorf("Healthcheck %s failed for %s: %s", b.id, endpointURL, err)
				} else {
					v.healthy = res.StatusCode < http.StatusInternalServerError // todo customize status code

					if !v.healthy {
						b.logger.Errorf("Endpoint %s is not healthy (%d). ID: %s", endpointURL, res.StatusCode, b.id)
					}
				}
				// v.healthy = true

				b.m.Lock()
				b.items[k] = v
				b.m.Unlock()
			}

		}
	}()
}
