package metrics

import (
	"os"
	"strconv"

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
	requestHost = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_host_total",
			Help: "Distribution by Request Host",
		},
		[]string{"hostname", "env", "host"},
	)
	httpMethods = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_http_methods_total",
			Help: "Distribution by HTTP methods",
		},
		[]string{"hostname", "env", "method"},
	)
	urlScheme = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_url_scheme_total",
			Help: "Distribution by URL scheme",
		},
		[]string{"hostname", "env", "scheme"},
	)
	requestSum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_sum_total",
			Help: "Total number of sent requests",
		},
		[]string{"hostname", "env"},
	)
	request1xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_1xx_total",
			Help: "Total number of sent 1xx requests",
		},
		[]string{"hostname", "env"},
	)
	request2xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_2xx_total",
			Help: "Total number of sent 2xx requests",
		},
		[]string{"hostname", "env"},
	)
	request3xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_3xx_total",
			Help: "Total number of sent 3xx requests",
		},
		[]string{"hostname", "env"},
	)
	request4xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_4xx_total",
			Help: "Total number of sent 4xx requests",
		},
		[]string{"hostname", "env"},
	)
	request5xx = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_request_5xx_total",
			Help: "Total number of sent 5xx requests",
		},
		[]string{"hostname", "env"},
	)
	hostHealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gpc_host_healthy",
			Help: "Health state of hosts",
		},
		[]string{"hostname", "env"},
	)
	hostUnhealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gpc_host_unhealthy",
			Help: "Health state of hosts",
		},
		[]string{"hostname", "env"},
	)
	cacheHit = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_cache_hits_total",
			Help: "The amount of cache hits",
		},
		[]string{"hostname", "env"},
	)
	cacheMiss = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_cache_miss_total",
			Help: "The amount of cache misses",
		},
		[]string{"hostname", "env"},
	)
	cacheStale = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gpc_cache_stale_total",
			Help: "The amount of cache misses",
		},
		[]string{"hostname", "env"},
	)
)

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

// Register - Add custom metric to prometheus.
func Register() {
	prometheus.MustRegister(
		statusCodes, requestSum,
		requestHost, httpMethods, urlScheme,
		request1xx, request2xx, request3xx, request4xx, request5xx,
		hostHealthy, hostUnhealthy,
		cacheHit, cacheMiss, cacheStale,
	)
}
