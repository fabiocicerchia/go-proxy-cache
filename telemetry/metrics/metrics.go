package metrics

import (
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

// baseLabels - Common "hostname"/"env" pair every metric is tagged with,
// merged with any metric-specific labels.
func baseLabels(extra prometheus.Labels) prometheus.Labels {
	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"env":      os.Getenv("TRACING_ENV"),
	}
	for k, v := range extra {
		labels[k] = v
	}
	return labels
}

// IncRequestHost - Increments metrics for gpc_request_host_total.
func IncRequestHost(host string) {
	requestHost.With(baseLabels(prometheus.Labels{"host": host})).Inc()
}

// IncHttpMethod - Increments metrics for gpc_http_methods_total.
func IncHttpMethod(method string) {
	httpMethods.With(baseLabels(prometheus.Labels{"method": method})).Inc()
}

// IncUrlScheme - Increments metrics for gpc_url_scheme_total.
func IncUrlScheme(scheme string) {
	urlScheme.With(baseLabels(prometheus.Labels{"scheme": scheme})).Inc()
}

// IncStatusCode - Increments metrics for gpc_status_codes_total, gpc_request_1xx_total, gpc_request_2xx_total, gpc_request_3xx_total, gpc_request_4xx_total, gpc_request_5xx_total, gpc_request_sum_total.
func IncStatusCode(code int) {
	statusCodes.With(baseLabels(prometheus.Labels{"code": strconv.Itoa(code)})).Inc()

	labels := baseLabels(nil)
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
func IncCacheMiss(server string) {
	cacheMiss.With(baseLabels(prometheus.Labels{"server": server})).Inc()
}

// IncCacheStale - Increments metrics for gpc_cache_stale_total.
func IncCacheStale(server string) {
	cacheStale.With(baseLabels(prometheus.Labels{"server": server})).Inc()
}

// IncCacheHit - Increments metrics for gpc_cache_hits_total.
func IncCacheHit(server string) {
	cacheHit.With(baseLabels(prometheus.Labels{"server": server})).Inc()
}

// SetHostHealthy - Increments metrics for gpc_host_healthy.
func SetHostHealthy(val float64) {
	hostHealthy.With(baseLabels(nil)).Set(val)
}

// SetHostUnhealthy - Increments metrics for gpc_host_unhealthy.
func SetHostUnhealthy(val float64) {
	hostUnhealthy.With(baseLabels(nil)).Set(val)
}
