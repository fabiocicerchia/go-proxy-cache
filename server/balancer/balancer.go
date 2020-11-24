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

var lb *roundrobin.Balancer

// InitRoundRobin - Initialise the LB algorithm for round robin.
func InitRoundRobin(endpoints []string) {
	lb = roundrobin.New(endpoints)
}

// GetLBRoundRobin - Returns backend server using RoundRobin algorithm.
func GetLBRoundRobin(defaultHost string) string {
	endpoint, err := lb.Pick()
	if err != nil || endpoint == "" {
		return defaultHost
	}

	return endpoint
}
