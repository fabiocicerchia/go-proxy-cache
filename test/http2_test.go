//go:build all || endtoend
// +build all endtoend

package test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"
)

func TestHTTP2ClientCall(t *testing.T) {
	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			DisableCompression: true,
			AllowHTTP:          false,
		},
	}

	req, err := http.NewRequest("GET", "https://testing.local:50443/", nil)
	assert.Nil(t, err)
	req.Host = "www.w3.org"
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	res.Body.Close()

	assert.NotEqual(t, "", res.Header.Get("X-Go-Proxy-Cache-Status"))

	assert.Equal(t, "HTTP/2.0", res.Proto)
	assert.Equal(t, 2, res.ProtoMajor)
	assert.Equal(t, 0, res.ProtoMinor)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, string(body), "<!doctype html>")
	assert.Contains(t, string(body), "<title>W3C</title>")
	assert.Contains(t, string(body), "</body>\n\n</html>\n")

	tearDownHttp2()
}

func TestHTTP2ClientCallToMissingDomain(t *testing.T) {
	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			DisableCompression: true,
			AllowHTTP:          false,
		},
	}

	req, err := http.NewRequest("GET", "https://testing.local:50443/", nil)
	assert.Nil(t, err)
	req.Host = "www.google.com"
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, "HTTP/2.0", res.Proto)
	assert.Equal(t, 2, res.ProtoMajor)
	assert.Equal(t, 0, res.ProtoMinor)

	assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
}

func tearDownHttp2() {
	req, _ := http.NewRequest("PURGE", "https://testing.local:50443/", nil)
	req.Host = "www.w3.org"
	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			DisableCompression: true,
			AllowHTTP:          false,
		},
	}
	_, _ = client.Do(req)
}
