package balancer

import (
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// TestHealthCheckInheritsTheUpstreamScheme - A health check probes the place
// the traffic goes. Defaulting it to https/443 independently of the upstream
// probed a plain-HTTP origin over TLS, and every check failed with "server
// gave HTTP response to HTTPS client".
func TestHealthCheckInheritsTheUpstreamScheme(t *testing.T) {
	http := config.Upstream{Scheme: "http", Port: "8080"}
	if hc := healthCheckFor(http); hc.Scheme != "http" || hc.Port != "8080" {
		t.Errorf("plain http upstream: got %s://:%s, want http://:8080", hc.Scheme, hc.Port)
	}

	// An explicit health-check setting still wins over the upstream.
	explicit := config.Upstream{Scheme: "http", Port: "8080"}
	explicit.HealthCheck = config.HealthCheck{Scheme: "https", Port: "8443"}
	if hc := healthCheckFor(explicit); hc.Scheme != "https" || hc.Port != "8443" {
		t.Errorf("explicit override: got %s://:%s, want https://:8443", hc.Scheme, hc.Port)
	}

	// Nothing declared anywhere keeps the old default.
	if hc := healthCheckFor(config.Upstream{}); hc.Scheme != "https" || hc.Port != "443" {
		t.Errorf("empty upstream: got %s://:%s, want https://:443", hc.Scheme, hc.Port)
	}

	// An https upstream on a non-standard port is probed there, not on 443.
	tls := config.Upstream{Scheme: "https", Port: "9443"}
	if hc := healthCheckFor(tls); hc.Port != "9443" {
		t.Errorf("https upstream port: got %s, want 9443", hc.Port)
	}
}
