//go:build all || endtoend
// +build all endtoend

package test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPClientCall(t *testing.T) {
	t.Skip("Found a regression due to an expected change in the endpoint")

	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://testing.local:50080/Consortium/", nil)
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

	assert.Equal(t, "MISS", res.Header.Get("X-Go-Proxy-Cache-Status"))

	assert.Equal(t, "HTTP/1.1", res.Proto)
	assert.Equal(t, 1, res.ProtoMajor)
	assert.Equal(t, 1, res.ProtoMinor)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, string(body), "<!DOCTYPE html PUBLIC")
	assert.Contains(t, string(body), `<title>About W3C</title>`)
	assert.Contains(t, string(body), "</div></body></html>\n")

	tearDownHttp()
}

func TestHTTPClientCallToMissingDomain(t *testing.T) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://testing.local:50080/", nil)
	assert.Nil(t, err)
	req.Host = "www.google.com"
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, "HTTP/1.1", res.Proto)
	assert.Equal(t, 1, res.ProtoMajor)
	assert.Equal(t, 1, res.ProtoMinor)

	assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
}

func tearDownHttp() {
	req, _ := http.NewRequest("PURGE", "http://testing.local:50080/Consortium/", nil)
	req.Host = "www.w3.org"
	client := &http.Client{}
	_, _ = client.Do(req)
}
