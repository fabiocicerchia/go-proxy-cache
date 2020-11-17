package balancer

import (
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer/roundrobin"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// GetLBRoundRobin - Returns backend server using RoundRobin algorithm.
func GetLBRoundRobin(endpoints []string, defaultHost string) string {
	lb := roundrobin.New([]interface{}{endpoints})
	endpoint, err := lb.Pick()
	if err != nil || utils.CastToString(endpoint) == "" {
		return defaultHost
	}

	return utils.CastToString(endpoint)
}
