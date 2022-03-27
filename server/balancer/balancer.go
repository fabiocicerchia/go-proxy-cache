package balancer

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

const lBIpHash = "ip-hash"
const lBLeastConnections = "least-connections"
const lBRandom = "random"
const lBRoundRobin = "round-robin"
const enableHealthchecks = true
const defaultClientTimeout = 5 * time.Second

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
		InitIpHash(name, config, enableHealthchecks)
	case lBLeastConnections:
		InitLeastConnection(name, config, enableHealthchecks)
	case lBRandom:
		InitRandom(name, config, enableHealthchecks)
	default: // round-robin (default)
		InitRoundRobin(name, config, enableHealthchecks)
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

// CheckHealth - Periodic check on nodes status.
func CheckHealth(b *NodeBalancer, config config.HealthCheck) {
	period := config.Interval
	if period == 0 {
		period = HealthCheckInterval
	}

	go func() {
		t := time.NewTicker(period)

		for {
			<-t.C

			healthyCounter := 0
			unhealthyCounter := 0
			for k, v := range b.Items {
				doHealthCheck(&v, config)

				if v.Healthy {
					healthyCounter++
				} else {
					unhealthyCounter++
				}

				b.M.Lock()
				b.Items[k] = v
				b.M.Unlock()
			}

			telemetry.RegisterHostHealth(healthyCounter, unhealthyCounter)
		}
	}()
}

func getClient(timeout time.Duration, tlsFlag bool, allowInsecure bool) *http.Client {
	if timeout == 0 {
		timeout = defaultClientTimeout
	}

	c := &http.Client{
		// return the 301/302
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: timeout,
	}

	if tlsFlag {
		c.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: allowInsecure,
			},
		}
	}

	return c
}

func doHealthCheck(v *Item, config config.HealthCheck) {
	url, _ := url.Parse(v.Endpoint)
	scheme := url.Scheme
	if scheme == "" || (scheme != "http" && scheme != "https") {
		scheme = config.Scheme
	}

	endpointURL := v.Endpoint
	if url.Scheme != scheme {
		endpointURL = fmt.Sprintf("%s://%s", scheme, v.Endpoint)
	}

	req, err := http.NewRequest("HEAD", endpointURL, nil)
	if err != nil {
		logger.GetGlobal().Errorf("Healthcheck request failed for %s: %s", endpointURL, err) // TODO: Add to trace span?
		return
	}
	res, err := getClient(config.Timeout, scheme == "https", config.AllowInsecure).Do(req)

	v.Healthy = err == nil
	if err != nil {
		logger.GetGlobal().Errorf("Healthcheck failed for %s: %s", endpointURL, err) // TODO: Add to trace span?
	} else {
		v.Healthy = slice.ContainsString(config.StatusCodes, strconv.Itoa(res.StatusCode))

		if !v.Healthy {
			logger.GetGlobal().Errorf("Endpoint %s is not healthy (%d).", endpointURL, res.StatusCode) // TODO: Add to trace span?
		}
	}
}

// GetHealthyNodes - Retrieves healthy nodes.
func (b *NodeBalancer) GetHealthyNodes() []Item {
	healthyNodes := []Item{}

	for _, v := range b.Items {
		if v.Healthy {
			healthyNodes = append(healthyNodes, v)
		}
	}

	return healthyNodes
}
