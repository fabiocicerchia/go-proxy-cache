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
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer/roundrobin"
)

type LoadBalancing map[string]*roundrobin.Balancer

var lb LoadBalancing

// InitRoundRobin - Initialise the LB algorithm for round robin.
func InitRoundRobin(name string, endpoints []string) {
	if len(lb) == 0 {
		lb = make(map[string]*roundrobin.Balancer)
	}

	lb[name] = roundrobin.New(name, endpoints, true)
}

// GetLBRoundRobin - Returns backend server using RoundRobin algorithm.
func GetLBRoundRobin(name string, defaultHost string) string {
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
