package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus namespaces and subsystems. They are part of the exported metric
// names (gpc_*, gpcee_upstream_*), so they are contract, not decoration.
const (
	namespace   = "gpc"
	namespaceEE = "gpcee"

	subsystemGeneric  = "generic"
	subsystemHTTP     = "http"
	subsystemUpstream = "upstream"
)

var (
	statusCodes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "status_codes_total",
			Help:      "Distribution by status codes",
		},
		[]string{"env", "hostname", "code"},
	)
	requestHost = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "request_host_total",
			Help:      "Distribution by Request Host",
		},
		[]string{"env", "hostname", "host"},
	)
	httpMethods = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_methods_total",
			Help:      "Distribution by HTTP methods",
		},
		[]string{"env", "hostname", "method"},
	)
	urlScheme = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "url_scheme_total",
			Help:      "Distribution by URL scheme",
		},
		[]string{"env", "hostname", "scheme"},
	)
	requestSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "request_sum_total",
			Help:      "Total number of sent requests",
		},
		[]string{"env", "hostname"},
	)
	request1xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "request_1xx_total",
			Help:      "Total number of sent 1xx requests",
		},
		[]string{"env", "hostname"},
	)
	request2xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "request_2xx_total",
			Help:      "Total number of sent 2xx requests",
		},
		[]string{"env", "hostname"},
	)
	request3xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "request_3xx_total",
			Help:      "Total number of sent 3xx requests",
		},
		[]string{"env", "hostname"},
	)
	request4xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "request_4xx_total",
			Help:      "Total number of sent 4xx requests",
		},
		[]string{"env", "hostname"},
	)
	request5xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "request_5xx_total",
			Help:      "Total number of sent 5xx requests",
		},
		[]string{"env", "hostname"},
	)
	hostHealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "host_healthy",
			Help:      "Health state of hosts",
		},
		[]string{"env", "hostname"},
	)
	hostUnhealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "host_unhealthy",
			Help:      "Health state of hosts",
		},
		[]string{"env", "hostname"},
	)
	cacheHit = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_hits_total",
			Help:      "The amount of cache hits",
		},
		[]string{"env", "hostname", "server"},
	)
	cacheMiss = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_miss_total",
			Help:      "The amount of cache misses",
		},
		[]string{"env", "hostname", "server"},
	)
	cacheStale = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_stale_total",
			Help:      "The amount of cache misses",
		},
		[]string{"env", "hostname", "server"},
	)

	// EE Metrics --------------------------------------------------------------
	gpceeBuildInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemGeneric,
			Name:      "build_info",
			Help:      "Shows the exporter build information.",
		}, []string{"env", "hostname", "git_commit", "version"},
	)
	gpceeUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemGeneric,
			Name:      "up",
			Help:      "Shows the status of the last metric scrape: `1` for a successful scrape and `0` for a failed one",
		}, []string{"env", "hostname"},
	)

	wholeRequest = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemHTTP,
			Name:      "request",
			Help:      "Distribution by Request",
		},
		[]string{"env", "hostname", "req_id", "url", "host", "scheme", "method", "protocol", "content_length"},
	)
	wholeResponse = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemHTTP,
			Name:      "response",
			Help:      "Distribution by Response",
		},
		[]string{"env", "hostname", "req_id", "url", "host", "scheme", "method", "protocol", "code", "cached", "stale", "size", "duration"},
	)
	gpceeHttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemHTTP,
			Name:      "requests_total",
			Help:      "Total http requests.",
		}, []string{"env", "hostname"},
	)

	gpceeUpstreamServerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemUpstream,
			Name:      "server_requests",
			Help:      "Total client requests.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerResponses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemUpstream,
			Name:      "server_responses",
			Help:      "Total responses sent to clients.",
		}, []string{"env", "hostname", "code", "server", "upstream"},
	)
	gpceeUpstreamServerSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemUpstream,
			Name:      "server_sent",
			Help:      "Bytes sent from this server.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemUpstream,
			Name:      "server_received",
			Help:      "Bytes received to this server.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerResponseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemUpstream,
			Name:      "server_response_time",
			Help:      "Total ms time to get the full response from the server.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerHealthChecksChecks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemUpstream,
			Name:      "server_health_checks_checks",
			Help:      "Total health check requests.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerHealthChecksFails = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemUpstream,
			Name:      "server_health_checks_fails",
			Help:      "Failed health checks.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerHealthChecksUnhealthy = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemUpstream,
			Name:      "server_health_checks_unhealthy",
			Help:      "How many times the server became unhealthy (state 'unhealthy').",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerHealthChecksStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespaceEE,
			Subsystem: subsystemUpstream,
			Name:      "server_health_checks_status",
			Help:      "Health server status.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
)

// Register - Add custom metric to prometheus.
func Register() {
	prometheus.MustRegister(
		statusCodes, requestSum,
		requestHost, httpMethods, urlScheme,
		request1xx, request2xx, request3xx, request4xx, request5xx,
		hostHealthy, hostUnhealthy,
		cacheHit, cacheMiss, cacheStale,

		// EE Metrics --------------------------------------------------------------
		wholeRequest, wholeResponse,
		gpceeBuildInfo, gpceeUp,
		gpceeHttpRequestsTotal,
		gpceeUpstreamServerRequests, gpceeUpstreamServerResponses, gpceeUpstreamServerSent,
		gpceeUpstreamServerReceived, gpceeUpstreamServerResponseTime,
		gpceeUpstreamServerHealthChecksChecks, gpceeUpstreamServerHealthChecksFails,
		gpceeUpstreamServerHealthChecksUnhealthy, gpceeUpstreamServerHealthChecksStatus,
	)
}
