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
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// InitRoundRobin - Initialise the LB algorithm for round robin.
func InitRoundRobin(name string, endpoints []string, enableHealthchecks bool) {
	if len(lb) == 0 {
		lb = make(LoadBalancing)
	}

	items := []Item{}
	for _, v := range endpoints {
		item := Item{Healthy: true, Endpoint: v}
		items = append(items, item)
	}

	lb[name] = NewRoundRobinBalancer(name, items)

	if enableHealthchecks {
		CheckHealth(lb[name].(*RoundRobinBalancer).Items, HealthCheckInterval) // todo customize
	}
}

// GetUpstreamNode - Returns backend server using current algorithm.
func GetUpstreamNode(name string, defaultHost string) string {
	var err error

	endpoint := ""

	if lbDomain, ok := lb[name]; ok {
		endpoint, err = lbDomain.Pick()
	}

	if err != nil || endpoint == "" {
		return defaultHost
	}

	return endpoint
}

// CheckHealth() - Period check on nodes status.
func CheckHealth(items []Item, period time.Duration) {
	go func() {
		t := time.NewTicker(period)

		for {
			<-t.C

			for k, v := range items {
				doHealthCheck(&v)

				// b.m.Lock()
				items[k] = v // TODO: CHECK IF BY REF
				// b.m.Unlock()
			}
		}
	}()
}

func getClient() *http.Client {
	return &http.Client{
		// return the 301/302
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second, // TODO: make it custom
	}
}

func doHealthCheck(v *Item) {
	endpointURL := fmt.Sprintf("http://%s", v.Endpoint) // todo fix

	req, err := http.NewRequest("GET", endpointURL, nil)
	if err != nil {
		log.Errorf("Healthcheck request failed for %s: %s", endpointURL, err)
		return
	}
	res, err := getClient().Do(req)

	v.Healthy = err == nil
	if err != nil {
		log.Errorf("Healthcheck failed for %s: %s", endpointURL, err)
	} else {
		v.Healthy = res.StatusCode < http.StatusInternalServerError // todo customize status code

		if !v.Healthy {
			log.Errorf("Endpoint %s is not healthy (%d).", endpointURL, res.StatusCode)
		}
	}
}
