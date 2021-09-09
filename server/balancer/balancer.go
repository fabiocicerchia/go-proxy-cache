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
	"net/url"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	log "github.com/sirupsen/logrus"
)

const lBIpHash = "ip-hash"
const lBLeastConnections = "least-connections"
const lBRandom = "random"
const lBRoundRobin = "round-robin"

func initLB() {
	if len(lb) == 0 {
		lb = make(LoadBalancing)
	}
}

func convertEndpoints(endpoints []string) []Item {
	items := []Item{}
	for _, v := range endpoints {
		item := Item{Healthy: true, Endpoint: v}
		items = append(items, item)
	}

	return items
}

// Init - Initialise the LB algorithm.
func Init(name string, config config.Upstream) {
	switch config.BalancingAlgorithm {
	case lBIpHash:
		InitIpHash(name, config, true)
	case lBLeastConnections:
		InitLeastConnection(name, config, true)
	case lBRandom:
		InitRandom(name, config, true)
	default: // round-robin (default)
		InitRoundRobin(name, config, true)
	}
}

// InitRoundRobin - Initialise the LB algorithm for round robin selection.
func InitRoundRobin(name string, config config.Upstream, enableHealthchecks bool) {
	initLB()
	items := convertEndpoints(config.Endpoints)

	lb[name] = NewRoundRobinBalancer(name, items)

	if enableHealthchecks {
		CheckHealth(&lb[name].(*RoundRobinBalancer).NodeBalancer, config.HealthCheck)
	}
}

// InitRandom - Initialise the LB algorithm for random selection.
func InitRandom(name string, config config.Upstream, enableHealthchecks bool) {
	initLB()
	items := convertEndpoints(config.Endpoints)

	lb[name] = NewRandomBalancer(name, items)

	if enableHealthchecks {
		CheckHealth(&lb[name].(*RandomBalancer).NodeBalancer, config.HealthCheck)
	}
}

// InitLeastConnection - Initialise the LB algorithm for least-connection selection.
func InitLeastConnection(name string, config config.Upstream, enableHealthchecks bool) {
	initLB()
	items := convertEndpoints(config.Endpoints)

	lb[name] = NewLeastConnectionsBalancer(name, items)

	if enableHealthchecks {
		CheckHealth(&lb[name].(*LeastConnectionsBalancer).NodeBalancer, config.HealthCheck)
	}
}

// InitIpHash - Initialise the LB algorithm for ip-hash selection.
func InitIpHash(name string, config config.Upstream, enableHealthchecks bool) {
	initLB()
	items := convertEndpoints(config.Endpoints)

	lb[name] = NewIpHashBalancer(name, items)

	if enableHealthchecks {
		CheckHealth(&lb[name].(*IpHashBalancer).NodeBalancer, config.HealthCheck)
	}
}

// GetUpstreamNode - Returns backend server using current algorithm.
func GetUpstreamNode(name string, requestURL url.URL, defaultHost string) string {
	var err error

	endpoint := ""

	if lbDomain, ok := lb[name]; ok {
		endpoint, err = lbDomain.Pick(requestURL.String())
	}

	if err != nil || endpoint == "" {
		return defaultHost
	}

	return endpoint
}

// CheckHealth() - Period check on nodes status.
func CheckHealth(b *NodeBalancer, config config.HealthCheck) {
	period := config.Interval
	if period == 0 {
		period = HealthCheckInterval
	}

	go func() {
		t := time.NewTicker(period)

		for {
			<-t.C

			for k, v := range b.Items {
				doHealthCheck(&v, config)
				// v.Healthy = false // TODO: CHECK IF BY REF

				b.M.Lock()
				b.Items[k] = v
				b.M.Unlock()
			}
		}
	}()
}

func getClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = 5 * time.Second // TODO: move to const
	}

	return &http.Client{
		// return the 301/302
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: timeout,
	}
}

// TODO: move to utils
func contains(s []int, val int) bool {
	for _, v := range s {
		if v == val {
			return true
		}
	}

	return false
}

func doHealthCheck(v *Item, config config.HealthCheck) {
	endpointURL := fmt.Sprintf("http://%s", v.Endpoint) // todo fix scheme

	req, err := http.NewRequest("HEAD", endpointURL, nil)
	if err != nil {
		log.Errorf("Healthcheck request failed for %s: %s", endpointURL, err)
		return
	}
	res, err := getClient(config.Timeout).Do(req)

	v.Healthy = err == nil
	if err != nil {
		log.Errorf("Healthcheck failed for %s: %s", endpointURL, err)
	} else {
		v.Healthy = contains(config.StatusCodes, res.StatusCode)

		if !v.Healthy {
			log.Errorf("Endpoint %s is not healthy (%d).", endpointURL, res.StatusCode)
		}
	}
}
