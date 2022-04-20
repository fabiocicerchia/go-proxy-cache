package metrics

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	statusCodes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "status_codes_total",
			Help:      "Distribution by status codes",
		},
		[]string{"env", "hostname", "code"},
	)
	requestHost = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "request_host_total",
			Help:      "Distribution by Request Host",
		},
		[]string{"env", "hostname", "host"},
	)
	httpMethods = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "http_methods_total",
			Help:      "Distribution by HTTP methods",
		},
		[]string{"env", "hostname", "method"},
	)
	urlScheme = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "url_scheme_total",
			Help:      "Distribution by URL scheme",
		},
		[]string{"env", "hostname", "scheme"},
	)
	requestSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "request_sum_total",
			Help:      "Total number of sent requests",
		},
		[]string{"env", "hostname"},
	)
	request1xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "request_1xx_total",
			Help:      "Total number of sent 1xx requests",
		},
		[]string{"env", "hostname"},
	)
	request2xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "request_2xx_total",
			Help:      "Total number of sent 2xx requests",
		},
		[]string{"env", "hostname"},
	)
	request3xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "request_3xx_total",
			Help:      "Total number of sent 3xx requests",
		},
		[]string{"env", "hostname"},
	)
	request4xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "request_4xx_total",
			Help:      "Total number of sent 4xx requests",
		},
		[]string{"env", "hostname"},
	)
	request5xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "request_5xx_total",
			Help:      "Total number of sent 5xx requests",
		},
		[]string{"env", "hostname"},
	)
	hostHealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gpc",
			Name:      "host_healthy",
			Help:      "Health state of hosts",
		},
		[]string{"env", "hostname"},
	)
	hostUnhealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gpc",
			Name:      "host_unhealthy",
			Help:      "Health state of hosts",
		},
		[]string{"env", "hostname"},
	)
	cacheHit = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "cache_hits_total",
			Help:      "The amount of cache hits",
		},
		[]string{"env", "hostname"},
	)
	cacheMiss = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "cache_miss_total",
			Help:      "The amount of cache misses",
		},
		[]string{"env", "hostname"},
	)
	cacheStale = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpc",
			Name:      "cache_stale_total",
			Help:      "The amount of cache misses",
		},
		[]string{"env", "hostname"},
	)

	// EE Metrics --------------------------------------------------------------
	gpceeBuildInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gpcee",
			Subsystem: "generic",
			Name:      "build_info",
			Help:      "Shows the exporter build information.",
		}, []string{"env", "hostname", "git_commit", "version"},
	)
	gpceeUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gpcee",
			Subsystem: "generic",
			Name:      "up",
			Help:      "Shows the status of the last metric scrape: `1` for a successful scrape and `0` for a failed one",
		}, []string{"env", "hostname"},
	)

	wholeRequest = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "http",
			Name:      "request",
			Help:      "Distribution by Request",
		},
		[]string{"env", "hostname", "req_id", "url", "host", "scheme", "method", "protocol", "content_length"},
	)
	wholeResponse = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "http",
			Name:      "response",
			Help:      "Distribution by Response",
		},
		[]string{"env", "hostname", "req_id", "url", "host", "scheme", "method", "protocol", "code", "cached", "stale", "size", "duration"},
	)
	gpceeHttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total http requests.",
		}, []string{"env", "hostname"},
	)

	gpceeUpstreamServerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "upstream",
			Name:      "server_requests",
			Help:      "Total client requests.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerResponses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "upstream",
			Name:      "server_responses",
			Help:      "Total responses sent to clients.",
		}, []string{"env", "hostname", "code", "server", "upstream"},
	)
	gpceeUpstreamServerSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "upstream",
			Name:      "server_sent",
			Help:      "Bytes sent from this server.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "upstream",
			Name:      "server_received",
			Help:      "Bytes received to this server.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerResponseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gpcee",
			Subsystem: "upstream",
			Name:      "server_response_time",
			Help:      "Total ms time to get the full response from the server.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerHealthChecksChecks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "upstream",
			Name:      "server_health_checks_checks",
			Help:      "Total health check requests.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerHealthChecksFails = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "upstream",
			Name:      "server_health_checks_fails",
			Help:      "Failed health checks.",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerHealthChecksUnhealthy = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gpcee",
			Subsystem: "upstream",
			Name:      "server_health_checks_unhealthy",
			Help:      "How many times the server became unhealthy (state 'unhealthy').",
		}, []string{"env", "hostname", "server", "upstream"},
	)
	gpceeUpstreamServerHealthChecksStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "gpcee",
			Subsystem: "upstream",
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

// IncRequestHost - Increments metrics for gpc_request_host_total.
func IncRequestHost(host string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"host":     host,
	}

	requestHost.With(labels).Inc()
}

// IncHttpMethod - Increments metrics for gpc_http_methods_total.
func IncHttpMethod(method string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"method":   method,
	}

	httpMethods.With(labels).Inc()
}

// IncUrlScheme - Increments metrics for gpc_url_scheme_total.
func IncUrlScheme(scheme string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"scheme":   scheme,
	}

	urlScheme.With(labels).Inc()
}

// IncStatusCode - Increments metrics for gpc_status_codes_total, gpc_request_1xx_total, gpc_request_2xx_total, gpc_request_3xx_total, gpc_request_4xx_total, gpc_request_5xx_total, gpc_request_sum_total.
func IncStatusCode(code int) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"code":     strconv.Itoa(code),
	}

	statusCodes.With(labels).Inc()

	labels = prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
	}
	if code < 200 {
		request1xx.With(labels).Inc()
	} else if code < 300 {
		request2xx.With(labels).Inc()
	} else if code < 400 {
		request3xx.With(labels).Inc()
	} else if code < 500 {
		request4xx.With(labels).Inc()
	} else if code < 600 {
		request5xx.With(labels).Inc()
	}

	requestSum.With(labels).Inc()
}

// IncCacheMiss - Increments metrics for gpc_cache_miss_total.
func IncCacheMiss() {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
	}

	cacheMiss.With(labels).Inc()
}

// IncCacheStale - Increments metrics for gpc_cache_stale_total.
func IncCacheStale() {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
	}

	cacheStale.With(labels).Inc()
}

// IncCacheHit - Increments metrics for gpc_cache_hits_total.
func IncCacheHit() {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
	}

	cacheHit.With(labels).Inc()
}

// SetHostHealthy - Increments metrics for gpc_host_healthy.
func SetHostHealthy(val float64) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
	}

	hostHealthy.With(labels).Set(val)
}

// SetHostUnhealthy - Increments metrics for gpc_host_unhealthy.
func SetHostUnhealthy(val float64) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
	}

	hostUnhealthy.With(labels).Set(val)
}

// EE Metrics ------------------------------------------------------------------

// IncWholeRequest - Increments metrics for gpcee_http_request_total.
func IncWholeRequest(reqID string, req http.Request, scheme string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname":       hostname,
		"env":            os.Getenv("TRACING_ENV"),
		"req_id":         reqID,
		"url":            req.URL.String(),
		"host":           req.Host,
		"scheme":         scheme,
		"method":         req.Method,
		"protocol":       req.Proto,
		"content_length": fmt.Sprintf("%d", req.ContentLength),
	}

	wholeRequest.With(labels).Inc()

	IncRequestHost(req.Host)
	IncHttpMethod(req.Method)
	IncUrlScheme(scheme)
	IncHttpRequestsTotal()
}

// IncWholeResponse - Increments metrics for gpcee_http_response_total.
func IncWholeResponse(reqID string, req http.Request, statusCode int, size int, duration int64, scheme string, cached bool, stale bool) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"host":     req.Host,
		"req_id":   reqID,
		"url":      req.URL.String(),
		"scheme":   scheme,
		"method":   req.Method,
		"protocol": req.Proto,
		"code":     fmt.Sprintf("%d", statusCode),
		"cached":   fmt.Sprintf("%v", cached),
		"stale":    fmt.Sprintf("%v", stale),
		"size":     fmt.Sprintf("%d", size),
		"duration": fmt.Sprintf("%d", duration),
	}

	wholeResponse.With(labels).Inc()
}

// SetBuildInfo - Set metrics for gpcee_build_info.
func SetBuildInfo(gitCommit string, version string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname":   hostname,
		"env":        os.Getenv("TRACING_ENV"),
		"git_commit": gitCommit,
		"version":    version,
	}

	gpceeBuildInfo.With(labels).Set(1)
}

// SetUp - Set metrics for gpcee_up.
func SetUp(val float64) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
	}
	gpceeUp.With(labels).Set(val)
}

// IncHttpRequestsTotal - Set metrics for gpcee_http_requests_total.
func IncHttpRequestsTotal() {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
	}

	gpceeHttpRequestsTotal.With(labels).Inc()
}

// IncUpstreamServerRequests - Set metrics for gpcee_upstream_server_requests.
func IncUpstreamServerRequests(server string, upstream string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerRequests.With(labels).Inc()
}

// IncUpstreamServerResponses - Set metrics for gpcee_upstream_server_responses.
func IncUpstreamServerResponses(code int, server string, upstream string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"code":     fmt.Sprintf("%dxx", code/100),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerResponses.With(labels).Inc()
}

// IncUpstreamServerSent - Increments metrics for gpcee_upstream_server_sent.
func IncUpstreamServerSent(server string, upstream string, val float64) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerSent.With(labels).Add(val)
}

// IncUpstreamServerReceived - Increments metrics for gpcee_upstream_server_received.
func IncUpstreamServerReceived(server string, upstream string, val float64) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerReceived.With(labels).Add(val)
}

// IncUpstreamServerResponseTime - Increment metrics for gpcee_upstream_server_response_time.
func IncUpstreamServerResponseTime(server string, upstream string, val float64) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}
	gpceeUpstreamServerResponseTime.With(labels).Add(val)
}

// IncUpstreamServerHealthChecksChecks - Increments metrics for gpcee_upstream_server_health_checks_checks.
func IncUpstreamServerHealthChecksChecks(server string, upstream string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerHealthChecksChecks.With(labels).Inc()
}

// IncUpstreamServerHealthChecksFails - Increments metrics for gpcee_upstream_server_health_checks_fails.
func IncUpstreamServerHealthChecksFails(server string, upstream string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerHealthChecksFails.With(labels).Inc()
}

// IncUpstreamServerHealthChecksUnhealthy - Increments metrics for gpcee_upstream_server_health_checks_unhealthy.
func IncUpstreamServerHealthChecksUnhealthy(server string, upstream string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerHealthChecksUnhealthy.With(labels).Inc()
}

// SetUpstreamServerHealthChecksHealthy - Set metrics for gpcee_upstream_server_health_checks_status.
func SetUpstreamServerHealthChecksHealthy(server string, upstream string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerHealthChecksStatus.With(labels).Set(1)

	IncUpstreamServerHealthChecksChecks(server, upstream)
}

// SetUpstreamServerHealthChecksFails - Set metrics for gpcee_upstream_server_health_checks_status.
func SetUpstreamServerHealthChecksFails(server string, upstream string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerHealthChecksStatus.With(labels).Set(-1)

	IncUpstreamServerHealthChecksFails(server, upstream)
}

// SetUpstreamServerHealthChecksUnhealthy - Set metrics for gpcee_upstream_server_health_checks_status.
func SetUpstreamServerHealthChecksUnhealthy(server string, upstream string) {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
		"server":   server,
		"upstream": upstream,
	}

	gpceeUpstreamServerHealthChecksStatus.With(labels).Set(0)

	IncUpstreamServerHealthChecksUnhealthy(server, upstream)
}
