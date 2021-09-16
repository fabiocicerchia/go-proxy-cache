package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	statusCodes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_status_codes_total",
			Help: "Distribution by status codes",
		},
		[]string{"hostname", "env", "code"},
	)
	requestSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_sum_total",
			Help: "Total number of sent requests",
		},
		[]string{},
	)
	request1xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_1xx_total",
			Help: "Total number of sent 1xx requests",
		},
		[]string{},
	)
	request2xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_2xx_total",
			Help: "Total number of sent 2xx requests",
		},
		[]string{},
	)
	request3xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_3xx_total",
			Help: "Total number of sent 3xx requests",
		},
		[]string{},
	)
	request4xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_4xx_total",
			Help: "Total number of sent 4xx requests",
		},
		[]string{},
	)
	request5xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_5xx_total",
			Help: "Total number of sent 5xx requests",
		},
		[]string{},
	)
	hostHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gpc_host_health",
			Help: "Health state of hosts by clusters",
		},
		[]string{},
	)
	cacheHit = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_cache_hits_total",
			Help: "The amount of cache hits",
		},
		[]string{},
	)
	cacheMiss = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_cache_miss_total",
			Help: "The amount of cache misses",
		},
		[]string{},
	)
	cacheStale = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_cache_stale_total",
			Help: "The amount of cache misses",
		},
		[]string{},
	)
)

func IncStatusCode(code string) {
	hostname, _ := os.Hostname()
	statusCodes.With(prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("ENV"),
		"code":     code,
	}).Inc()
}

func Register() {
	prometheus.MustRegister(
		statusCodes, requestSum,
		request1xx, request2xx, request3xx, request4xx, request5xx,
		hostHealth,
		cacheHit, cacheMiss, cacheStale,
	)

	IncStatusCode("200")
}
