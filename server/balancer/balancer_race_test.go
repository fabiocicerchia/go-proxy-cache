//go:build all || unit
// +build all unit

package balancer_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"fmt"
	"sync"
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
)

func testItems() []balancer.Item {
	return []balancer.Item{
		{Endpoint: "item1", Healthy: true},
		{Endpoint: "item2", Healthy: true},
		{Endpoint: "item3", Healthy: true},
	}
}

// TestConcurrentPickWithHealthFlips - Ensures Pick() is safe while node health
// is being updated concurrently (as the healthcheck goroutine does).
// Run with -race to be meaningful.
func TestConcurrentPickWithHealthFlips(t *testing.T) {
	initLogs()

	type instance struct {
		balancer.Balancer
		node *balancer.NodeBalancer
	}

	rr := balancer.NewRoundRobinBalancer("race-rr", testItems())
	rnd := balancer.NewRandomBalancer("race-rnd", testItems())
	iph := balancer.NewIpHashBalancer("race-iph", testItems())
	lc := balancer.NewLeastConnectionsBalancer("race-lc", testItems())

	instances := []instance{
		{rr, &rr.NodeBalancer},
		{rnd, &rnd.NodeBalancer},
		{iph, &iph.NodeBalancer},
		{lc, &lc.NodeBalancer},
	}

	for _, inst := range instances {
		inst := inst

		var wg sync.WaitGroup

		// Simulates the healthcheck goroutine flipping node health.
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				inst.node.M.Lock()
				inst.node.Items[i%len(inst.node.Items)].Healthy = i%2 == 0
				inst.node.M.Unlock()
			}
		}()

		// Concurrent pickers.
		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				for i := 0; i < 500; i++ {
					_, _ = inst.Pick(fmt.Sprintf("https://example.com/%d/%d", g, i))
				}
			}(g)
		}

		wg.Wait()
	}
}
