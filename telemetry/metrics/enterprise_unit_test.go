//go:build all || unit
// +build all unit

package metrics

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every request used to mint a new series (req_id, url, size, duration
// labels), so memory grew without bound and /metrics listed every URL.
func TestWholeRequestResponseSeriesAreBounded(t *testing.T) {
	wholeRequest.Reset()
	wholeResponse.Reset()

	for i := 0; i < 50; i++ {
		u, _ := url.Parse(fmt.Sprintf("/page/%d?token=secret%d", i, i))
		req := http.Request{Method: "GET", URL: u, Host: fmt.Sprintf("example.com:%d", 1000+i), Proto: "HTTP/1.1", ContentLength: int64(i)}

		IncWholeRequest(req, "https")
		IncWholeResponse(req, 200, "https", true, false)
	}

	reg := prometheus.NewPedanticRegistry()
	require.NoError(t, reg.Register(wholeRequest))
	require.NoError(t, reg.Register(wholeResponse))
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 2)

	for _, mf := range families {
		assert.Len(t, mf.GetMetric(), 1, "%s must keep one series however many requests arrive", mf.GetName())
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				assert.NotContains(t, l.GetValue(), "secret", "%s{%s}", mf.GetName(), l.GetName())
				if l.GetName() == "host" {
					assert.Equal(t, "example.com", l.GetValue())
				}
			}
		}
	}
}
