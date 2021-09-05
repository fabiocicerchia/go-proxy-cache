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
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestETagValidResponse(t *testing.T) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://testing.local:50080/", nil)
	// Need to fetch fresh content to verify the ETag.
	req.Header = http.Header{
		"X-Go-Proxy-Cache-Force-Fresh": []string{"1"},
	}
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
	// this is the real ETag from w3.org
	assert.Regexp(t, regexp.MustCompile(`^\"[0-9a-f]{4}-[0-9a-f]{13};[0-9a-f]{2}-[0-9a-f]{13}-gzip\"$`), res.Header.Get("ETag"))

	assert.Equal(t, "HTTP/1.1", res.Proto)
	assert.Equal(t, 1, res.ProtoMajor)
	assert.Equal(t, 1, res.ProtoMinor)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, string(body), "<!DOCTYPE html PUBLIC")
	assert.Contains(t, string(body), `<title>World Wide Web Consortium (W3C)</title>`)
	assert.Contains(t, string(body), "</body>\n</html>\n")

	tearDownETag()
}

func TestETagIfModifiedSinceWhenChanged(t *testing.T) {
	client := &http.Client{}

	today := "Thu, 01 Jan 1970 00:00:00 GMT"

	req, err := http.NewRequest("GET", "http://testing.local:50080/etag", nil)
	assert.Nil(t, err)
	req.Host = "testing.local"
	req.Header = http.Header{
		"If-Modified-Since": []string{today},
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	res.Body.Close()

	assert.Equal(t, "HTTP/1.1", res.Proto)
	assert.Equal(t, 1, res.ProtoMajor)
	assert.Equal(t, 1, res.ProtoMinor)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.NotEqual(t, []byte{}, body)
}

func TestETagIfModifiedSinceWhenNotChanged(t *testing.T) {
	client := &http.Client{}

	today := "Thu, 01 Jan 1970 00:00:01 GMT"

	req, err := http.NewRequest("GET", "http://testing.local:50080/etag", nil)
	assert.Nil(t, err)
	req.Host = "testing.local"
	req.Header = http.Header{
		"If-Modified-Since": []string{today},
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	res.Body.Close()

	assert.Equal(t, "HTTP/1.1", res.Proto)
	assert.Equal(t, 1, res.ProtoMajor)
	assert.Equal(t, 1, res.ProtoMinor)

	assert.Equal(t, http.StatusNotModified, res.StatusCode)
	assert.Equal(t, []byte{}, body)
}

func TestETagIfUnmodifiedSince(t *testing.T) {
	t.Skip("Need to be implemented.")
}

func TestETagIfNoneMatchAsMatch(t *testing.T) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://testing.local:50080/etag", nil)
	assert.Nil(t, err)
	req.Host = "testing.local"
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	res.Body.Close()

	etag := res.Header.Get("Etag")

	assert.Equal(t, "HTTP/1.1", res.Proto)
	assert.Equal(t, 1, res.ProtoMajor)
	assert.Equal(t, 1, res.ProtoMinor)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.NotNil(t, body)

	// -------------------------------------------------------------------------

	req, err = http.NewRequest("GET", "http://testing.local:50080/etag", nil)
	assert.Nil(t, err)
	req.Host = "testing.local"
	req.Header = http.Header{
		"If-None-Match": []string{etag},
	}
	res, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	res.Body.Close()

	assert.Equal(t, "HTTP/1.1", res.Proto)
	assert.Equal(t, 1, res.ProtoMajor)
	assert.Equal(t, 1, res.ProtoMinor)

	assert.Equal(t, http.StatusNotModified, res.StatusCode)
	assert.Equal(t, []byte{}, body)
}

func TestETagIfNoneMatchAsNotMatch(t *testing.T) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://testing.local:50080/etag", nil)
	assert.Nil(t, err)
	req.Host = "testing.local"
	req.Header = http.Header{
		"If-None-Match": []string{"12345-qwerty"},
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	res.Body.Close()

	assert.Equal(t, "HTTP/1.1", res.Proto)
	assert.Equal(t, 1, res.ProtoMajor)
	assert.Equal(t, 1, res.ProtoMinor)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.NotNil(t, body)
}

func TestETagIfMatchAsMatch(t *testing.T) {
	t.Skip("Need to be implemented.")
}

func TestETagIfMatchAsNotMatch(t *testing.T) {
	t.Skip("Need to be implemented.")
}

func tearDownETag() {
	req, _ := http.NewRequest("PURGE", "http://testing.local:50080/", nil)
	req.Host = "www.w3.org"
	client := &http.Client{}
	_, _ = client.Do(req)
}
