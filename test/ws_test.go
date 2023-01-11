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
	"net/url"
	"os"
	"os/signal"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestEndToEndWebSocket(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{
		Scheme: "ws",
		Host:   "testing.local:50080",
		Path:   "/",
	}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.Nil(t, err)
	defer c.Close()

	err = c.WriteMessage(websocket.TextMessage, []byte("test message"))
	assert.Nil(t, err)

	_, message, err := c.ReadMessage()
	assert.Nil(t, err)
	assert.Equal(t, "Server received from client: test message", string(message))
}

func TestEndToEndWebSocketSecure(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{
		Scheme: "wss",
		Host:   "testing.local:50443",
		Path:   "/",
	}

	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	c, _, err := dialer.Dial(u.String(), nil)
	assert.Nil(t, err)
	defer c.Close()

	err = c.WriteMessage(websocket.TextMessage, []byte("test message"))
	assert.Nil(t, err)

	_, message, err := c.ReadMessage()
	assert.Nil(t, err)
	assert.Equal(t, "Server received from client: test message", string(message))
}
