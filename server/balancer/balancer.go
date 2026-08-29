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
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

const lBIpHash = "ip-hash"
const lBLeastConnections = "least-connections"
const lBRandom = "random"
const lBRoundRobin = "round-robin"
const enableHealthchecks = true
const defaultClientTimeout = 5 * time.Second

func initLB() {
	if lb == nil {
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
	case lBRoundRobin:
		InitRoundRobin(name, config, enableHealthchecks)
	default: // round-robin (default)
		InitRoundRobin(name, config, enableHealthchecks)
	}
}

// initBalancer - Shared init for every LB algorithm: build the items, store
// the balancer, and health-check it. Only the constructor differs per algorithm.
func initBalancer(name string, config config.Upstream, enableHealthchecks bool, newBalancer func(string, []Item) Balancer) {
	items := convertEndpoints(config.Endpoints)
	b := newBalancer(name, items)

	lbMu.Lock()
	initLB()

	// Replacing a balancer must stop the health-check goroutine of the one it
	// replaces, otherwise every reconfiguration leaks a ticker.
	if stop, ok := stopHealthChecks[name]; ok {
		close(stop)
		delete(stopHealthChecks, name)
	}

	lb[name] = b

	var stop chan struct{}
	if enableHealthchecks {
		stop = make(chan struct{})
		stopHealthChecks[name] = stop
	}
	lbMu.Unlock()

	if enableHealthchecks {
		CheckHealth(b.GetNodeBalancer(), config.Host, config.HealthCheck, stop)
	}
}

// Remove - Drops a balancer and stops its health-check goroutine.
func Remove(name string) {
	lbMu.Lock()
	defer lbMu.Unlock()

	if stop, ok := stopHealthChecks[name]; ok {
		close(stop)
		delete(stopHealthChecks, name)
	}

	delete(lb, name)
}

// InitRoundRobin - Initialise the LB algorithm for round robin selection.
func InitRoundRobin(name string, config config.Upstream, enableHealthchecks bool) {
	initBalancer(name, config, enableHealthchecks, func(n string, items []Item) Balancer { return NewRoundRobinBalancer(n, items) })
}

// InitRandom - Initialise the LB algorithm for random selection.
func InitRandom(name string, config config.Upstream, enableHealthchecks bool) {
	initBalancer(name, config, enableHealthchecks, func(n string, items []Item) Balancer { return NewRandomBalancer(n, items) })
}

// InitLeastConnection - Initialise the LB algorithm for least-connection selection.
func InitLeastConnection(name string, config config.Upstream, enableHealthchecks bool) {
	initBalancer(name, config, enableHealthchecks, func(n string, items []Item) Balancer { return NewLeastConnectionsBalancer(n, items) })
}

// InitIpHash - Initialise the LB algorithm for ip-hash selection.
func InitIpHash(name string, config config.Upstream, enableHealthchecks bool) {
	initBalancer(name, config, enableHealthchecks, func(n string, items []Item) Balancer { return NewIpHashBalancer(n, items) })
}

// GetUpstreamNode - Returns backend server using current algorithm.
func GetUpstreamNode(name string, requestURL url.URL, defaultHost string) string {
	var err error

	endpoint := ""

	lbMu.RLock()
	lbDomain, ok := lb[name]
	lbMu.RUnlock()

	if ok {
		endpoint, err = lbDomain.Pick(requestURL.String())
	}

	if err != nil || endpoint == "" {
		return defaultHost
	}

	return endpoint
}

// CheckHealth - Periodic check on nodes status.
//
// The stop channel terminates the goroutine when the balancer is replaced or
// removed. Pass nil for a balancer that lives for the whole process lifetime.
func CheckHealth(b *NodeBalancer, host string, config config.HealthCheck, stop <-chan struct{}) {
	period := config.Interval
	if period == 0 {
		period = HealthCheckInterval
	}

	go func() {
		t := time.NewTicker(period)
		defer t.Stop()

		for {
			select {
			case <-stop:
				return
			case <-t.C:
			}

			// Work on a snapshot: iterating b.Items unlocked while Pick()
			// goroutines read it (and this loop writes it) is a data race.
			b.M.RLock()
			items := make([]Item, len(b.Items))
			copy(items, b.Items)
			b.M.RUnlock()

			healthyCounter := 0
			unhealthyCounter := 0

			for k := range items {
				DoHealthCheck(&items[k], host, config)

				if items[k].Healthy {
					healthyCounter++
				} else {
					unhealthyCounter++
				}
			}

			b.M.Lock()
			for k := range items {
				if k < len(b.Items) {
					b.Items[k] = items[k]
				}
			}
			b.M.Unlock()

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
		} //#nosec G402
	}

	return c
}

func DoHealthCheck(v *Item, host string, config config.HealthCheck) {
	url, _ := url.Parse(v.Endpoint)
	scheme := url.Scheme
	if scheme == "" || (scheme != "http" && scheme != "https") {
		scheme = config.Scheme
	}

	hostWithPort := url.Host
	if hostWithPort == "" {
		hostWithPort = v.Endpoint
	}
	_, port, err := net.SplitHostPort(hostWithPort)

	overridePort := ""
	if err != nil || port == "" {
		overridePort = fmt.Sprintf(":%s", config.Port)
	}

	overrideScheme := ""
	if url.Scheme != scheme {
		overrideScheme = fmt.Sprintf("%s://", scheme)
	}

	endpointURL := fmt.Sprintf("%s%s%s", overrideScheme, v.Endpoint, overridePort)

	req, err := http.NewRequest("HEAD", endpointURL, nil)
	if err != nil {
		logger.GetGlobal().Errorf("Healthcheck request failed for %s / %s: %s", host, endpointURL, err) // TODO: Add to trace span?
		return
	}
	res, err := getClient(config.Timeout, scheme == "https", config.AllowInsecure).Do(req)

	v.Healthy = err == nil
	if err != nil {
		logger.GetGlobal().Errorf("Healthcheck failed for %s: %s", endpointURL, err) // TODO: Add to trace span?
		metrics.SetUpstreamServerHealthChecksFails(host, hostWithPort)
	} else {
		// The body must always be closed, even for HEAD responses, or the
		// underlying connection is never released: one leaked connection per
		// node per healthcheck interval, forever.
		defer res.Body.Close()
		v.Healthy = slice.ContainsString(config.StatusCodes, strconv.Itoa(res.StatusCode))

		if !v.Healthy {
			logger.GetGlobal().Errorf("Endpoint %s is not healthy (%d).", endpointURL, res.StatusCode) // TODO: Add to trace span?
			metrics.SetUpstreamServerHealthChecksUnhealthy(host, hostWithPort)
		} else {
			metrics.SetUpstreamServerHealthChecksHealthy(host, hostWithPort)
		}
	}
}

// GetHealthyNodes - Retrieves healthy nodes.
// It locks internally: callers must NOT hold b.M when invoking it (nested
// RLock acquisition can deadlock with a pending writer).
func (b *NodeBalancer) GetHealthyNodes() []Item {
	healthyNodes := []Item{}

	b.M.RLock()
	defer b.M.RUnlock()

	for _, v := range b.Items {
		if v.Healthy {
			healthyNodes = append(healthyNodes, v)
		}
	}

	return healthyNodes
}
