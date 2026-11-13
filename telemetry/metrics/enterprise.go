package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// EE Metrics ------------------------------------------------------------------

// detailedRequestSeries - Whether the per-request series are recorded.
//
// gpcee_http_request and gpcee_http_response add a label combination per
// host/scheme/method/protocol (and code/cached/stale) on top of the aggregate
// counters, which are always recorded. Bounded, but it can still be switched
// off when those series are not wanted (e.g. one host per Ingress rule).
//
// On by default, so existing dashboards keep working.
var detailedRequestSeries atomic.Bool

func init() {
	detailedRequestSeries.Store(true)
}

// SetDetailedRequestSeries - Enables or disables the per-request series.
func SetDetailedRequestSeries(enabled bool) {
	detailedRequestSeries.Store(enabled)
}

// IncWholeRequest - Increments metrics for gpcee_http_request_total.
//
// Labels are limited to values with a small, fixed set: anything per request
// (request ID, URL, sizes, durations) would create a new series on every call
// and leak the requested URLs through /metrics. The port is stripped from the
// client-supplied Host header for the same reason.
func IncWholeRequest(req http.Request, scheme string) {
	host := utils.StripPort(req.Host)
	method := boundedMethod(req.Method)

	if !detailedRequestSeries.Load() {
		IncRequestHost(host)
		IncHttpMethod(method)
		IncUrlScheme(scheme)
		IncHttpRequestsTotal()

		return
	}

	wholeRequest.With(baseLabels(prometheus.Labels{
		"host":     host,
		"scheme":   scheme,
		"method":   method,
		"protocol": req.Proto,
	})).Inc()

	IncRequestHost(host)
	IncHttpMethod(method)
	IncUrlScheme(scheme)
	IncHttpRequestsTotal()
}

// IncWholeResponse - Increments metrics for gpcee_http_response_total.
// See IncWholeRequest for why the label set is kept bounded.
func IncWholeResponse(req http.Request, statusCode int, scheme string, cached bool, stale bool) {
	if !detailedRequestSeries.Load() {
		return
	}

	wholeResponse.With(baseLabels(prometheus.Labels{
		"host":     utils.StripPort(req.Host),
		"scheme":   scheme,
		"method":   boundedMethod(req.Method),
		"protocol": req.Proto,
		"code":     fmt.Sprintf("%d", statusCode),
		"cached":   fmt.Sprintf("%v", cached),
		"stale":    fmt.Sprintf("%v", stale),
	})).Inc()
}

// boundedMethod - net/http accepts any token as a method, so anything outside
// the standard set (plus PURGE) is folded into "OTHER" before it becomes a
// label value.
func boundedMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace, "PURGE":
		return method
	}

	return "OTHER"
}

// SetBuildInfo - Set metrics for gpcee_build_info.
func SetBuildInfo(gitCommit string, version string) {
	gpceeBuildInfo.With(baseLabels(prometheus.Labels{
		"git_commit": gitCommit,
		"version":    version,
	})).Set(1)
}

// SetUp - Set metrics for gpcee_up.
func SetUp(val float64) {
	gpceeUp.With(baseLabels(nil)).Set(val)
}

// IncHttpRequestsTotal - Set metrics for gpcee_http_requests_total.
func IncHttpRequestsTotal() {
	gpceeHttpRequestsTotal.With(baseLabels(nil)).Inc()
}

// IncUpstreamServerRequests - Set metrics for gpcee_upstream_server_requests.
func IncUpstreamServerRequests(server string, upstream string) {
	gpceeUpstreamServerRequests.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Inc()
}

// IncUpstreamServerResponses - Set metrics for gpcee_upstream_server_responses.
func IncUpstreamServerResponses(code int, server string, upstream string) {
	gpceeUpstreamServerResponses.With(baseLabels(prometheus.Labels{
		"code":     fmt.Sprintf("%dxx", code/100),
		"server":   server,
		"upstream": upstream,
	})).Inc()
}

// IncUpstreamServerSent - Increments metrics for gpcee_upstream_server_sent.
func IncUpstreamServerSent(server string, upstream string, val float64) {
	gpceeUpstreamServerSent.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Add(val)
}

// IncUpstreamServerReceived - Increments metrics for gpcee_upstream_server_received.
func IncUpstreamServerReceived(server string, upstream string, val float64) {
	gpceeUpstreamServerReceived.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Add(val)
}

// IncUpstreamServerResponseTime - Increment metrics for gpcee_upstream_server_response_time.
func IncUpstreamServerResponseTime(server string, upstream string, val float64) {
	gpceeUpstreamServerResponseTime.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Add(val)
}

// IncUpstreamServerHealthChecksChecks - Increments metrics for gpcee_upstream_server_health_checks_checks.
func IncUpstreamServerHealthChecksChecks(server string, upstream string) {
	gpceeUpstreamServerHealthChecksChecks.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Inc()
}

// IncUpstreamServerHealthChecksFails - Increments metrics for gpcee_upstream_server_health_checks_fails.
func IncUpstreamServerHealthChecksFails(server string, upstream string) {
	gpceeUpstreamServerHealthChecksFails.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Inc()
}

// IncUpstreamServerHealthChecksUnhealthy - Increments metrics for gpcee_upstream_server_health_checks_unhealthy.
func IncUpstreamServerHealthChecksUnhealthy(server string, upstream string) {
	gpceeUpstreamServerHealthChecksUnhealthy.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Inc()
}

// SetUpstreamServerHealthChecksHealthy - Set metrics for gpcee_upstream_server_health_checks_status.
func SetUpstreamServerHealthChecksHealthy(server string, upstream string) {
	gpceeUpstreamServerHealthChecksStatus.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Set(1)
	IncUpstreamServerHealthChecksChecks(server, upstream)
}

// SetUpstreamServerHealthChecksFails - Set metrics for gpcee_upstream_server_health_checks_status.
func SetUpstreamServerHealthChecksFails(server string, upstream string) {
	gpceeUpstreamServerHealthChecksStatus.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Set(-1)
	IncUpstreamServerHealthChecksFails(server, upstream)
}

// SetUpstreamServerHealthChecksUnhealthy - Set metrics for gpcee_upstream_server_health_checks_status.
func SetUpstreamServerHealthChecksUnhealthy(server string, upstream string) {
	gpceeUpstreamServerHealthChecksStatus.With(baseLabels(prometheus.Labels{"server": server, "upstream": upstream})).Set(0)
	IncUpstreamServerHealthChecksUnhealthy(server, upstream)
}
