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
	"sync"
	"time"
)

// TODO: make it customizable
const HealthCheckInterval time.Duration = 30 * time.Second

type LoadBalancing map[string]Balancer

var lb LoadBalancing

type Item struct {
	Healthy  bool
	Endpoint string
}

type NodeBalancer struct {
	M sync.Mutex

	Id    string
	Items []Item
}

// Balancer instance.
type Balancer interface {
	GetHealthyNodes() []Item
	Pick() (string, error)
}
