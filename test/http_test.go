// +build endtoend

package test

import (
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPClientCall(t *testing.T) {
	client := &http.Client{}

	res, err := client.Get("http://localhost")
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
	assert.Contains(t, string(body), "<!DOCTYPE html>\n<html lang=\"en\"")
	assert.Contains(t, string(body), `<title>Fabio Cicerchia`)
	assert.Contains(t, string(body), "</body>\n</html>")
}
