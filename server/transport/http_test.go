//go:build all || unit
// +build all unit

package transport_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
)

// A cached response must reach the client with the status code it was stored
// with, not the implicit 200 net/http sends when the body is written first.
func TestServeCachedResponseSendsCachedStatusCode(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		header  string
		value   string
		content [][]byte
	}{
		{"200", http.StatusOK, "Content-Type", "text/plain", [][]byte{[]byte("ok")}},
		{"301", http.StatusMovedPermanently, "Location", "https://example.com/new", [][]byte{[]byte("moved")}},
		{"302", http.StatusFound, "Location", "https://example.com/tmp", [][]byte{[]byte("found")}},
		{"404", http.StatusNotFound, "Content-Type", "text/html", [][]byte{[]byte("not "), []byte("found")}},
		{"500", http.StatusInternalServerError, "Content-Type", "text/plain", [][]byte{[]byte("boom")}},
		{"301 empty body", http.StatusMovedPermanently, "Location", "https://example.com/empty", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			lwr := response.NewLoggedResponseWriter(rec, "test-request-id")

			transport.ServeCachedResponse(context.Background(), lwr, cache.URIObj{
				StatusCode:      tt.status,
				ResponseHeaders: http.Header{tt.header: []string{tt.value}},
				Content:         tt.content,
			})

			want := []byte{}
			for _, c := range tt.content {
				want = append(want, c...)
			}

			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, string(want), rec.Body.String())
			assert.Equal(t, tt.value, rec.Header().Get(tt.header))
		})
	}
}
