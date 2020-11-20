package balancer

import (
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer/roundrobin"
)

var lb *roundrobin.Balancer

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
