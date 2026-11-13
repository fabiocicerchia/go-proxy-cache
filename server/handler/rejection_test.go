package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// TestRejectionReasonNamesTheCheckThatFailed - A 501 has to say which of the
// three checks turned the request away: "configuration mismatch" was true of
// all of them and actionable for none.
func TestRejectionReasonNamesTheCheckThatFailed(t *testing.T) {
	rc := RequestCall{Request: http.Request{Host: "frankenstein.local"}}

	if got := rejectionReason(rc, false, "80"); !strings.Contains(got, "no domain in the config matches") {
		t.Errorf("unmatched domain: %q", got)
	}

	rc.DomainConfig = config.Configuration{}
	rc.DomainConfig.Server.Upstream.Host = "api"
	rc.DomainConfig.Server.Port = config.Port{HTTP: "80"}
	got := rejectionReason(rc, true, "80")
	if !strings.Contains(got, "server.upstream.host") || !strings.Contains(got, "endpoints") {
		t.Errorf("host mismatch should name the field and the way out: %q", got)
	}

	rc.DomainConfig.Server.Upstream.Host = "frankenstein.local"
	if got := rejectionReason(rc, true, "8082"); !strings.Contains(got, ":8082") {
		t.Errorf("port mismatch should name the port it arrived on: %q", got)
	}
}
