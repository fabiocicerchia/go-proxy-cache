// +build all endtoend

package test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
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
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
		return
	}

	res.Body.Close()

	assert.Equal(t, "MISS", res.Header.Get("X-Go-Proxy-Cache-Status"))

	assert.Equal(t, "HTTP/2.0", res.Proto)
	assert.Equal(t, 2, res.ProtoMajor)
	assert.Equal(t, 0, res.ProtoMinor)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, string(body), "<!DOCTYPE html PUBLIC")
	assert.Contains(t, string(body), `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, string(body), "</body>\n</html>\n")
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
		return
	}

	assert.Equal(t, "HTTP/2.0", res.Proto)
	assert.Equal(t, 2, res.ProtoMajor)
	assert.Equal(t, 0, res.ProtoMinor)

	assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
}
