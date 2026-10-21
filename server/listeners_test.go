//go:build all || unit
// +build all unit

package server

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
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A fresh connection per request: each one must be accepted, not reuse a keep-alive.
var testClient = &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}

func get(t *testing.T, url string) string {
	res, err := testClient.Get(url)
	if err != nil {
		t.Errorf("GET %s: %s", url, err)
		return ""
	}
	defer func() { _ = res.Body.Close() }()

	body, _ := io.ReadAll(res.Body)

	return string(body)
}

func listenLocal(t *testing.T) (net.Listener, string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	return ln, strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

// TestHelperProcess - The go-proxy-cache side of TestInheritedListenerSurvivesReexec.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GPC_TEST_HELPER") != "1" {
		t.Skip("run by TestInheritedListenerSurvivesReexec")
	}

	gen := "systemd"
	if os.Getenv(envReexecFDs) != "" {
		gen = "reexec"
	}

	inherited, err := inheritedListeners()
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "%s %d", gen, os.Getpid())
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = io.WriteString(w, "slow")
	})

	port := os.Getenv("GPC_TEST_PORT")
	s := &Servers{HTTP: map[string]*Server{port: {HttpSrv: &http.Server{Handler: mux}}}, inherited: inherited}
	s.startListeners()
	s.waitForSignal(context.Background(), 5*time.Second)
}

func TestInheritedListenerSurvivesReexec(t *testing.T) {
	ln, port := listenLocal(t)
	f, err := ln.(*net.TCPListener).File()
	require.NoError(t, err)

	// fd 3 in the child, as systemd socket activation passes it.
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), "GPC_TEST_HELPER=1", "GPC_TEST_PORT="+port, "LISTEN_FDS=1")
	cmd.ExtraFiles = []*os.File{f}
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	_ = f.Close()
	_ = ln.Close()

	defer func() { _ = cmd.Process.Kill() }()

	url := "http://" + ln.Addr().String()
	pid := cmd.Process.Pid

	assert.Equal(t, fmt.Sprintf("systemd %d", pid), get(t, url+"/"))
	time.Sleep(200 * time.Millisecond) // let it reach signal.Notify

	slow := make(chan string)
	go func() { slow <- get(t, url+"/slow") }()
	time.Sleep(100 * time.Millisecond)

	require.NoError(t, cmd.Process.Signal(syscall.SIGHUP))

	// No request may fail across the drain and the exec; they queue on the socket.
	deadline := time.Now().Add(10 * time.Second)
	for get(t, url+"/") != fmt.Sprintf("reexec %d", pid) {
		require.False(t, t.Failed() || time.Now().After(deadline), "never served by the re-executed process")
	}

	assert.Equal(t, "slow", <-slow, "in-flight request must finish before the exec")

	time.Sleep(200 * time.Millisecond)
	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))
	require.NoError(t, cmd.Wait())
}

func TestShutdownDrainsInFlightRequests(t *testing.T) {
	ln, port := listenLocal(t)

	entered, release := make(chan struct{}), make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = io.WriteString(w, "done")
	})

	s := &Servers{
		HTTP:      map[string]*Server{port: {HttpSrv: &http.Server{Handler: handler}}},
		inherited: map[string]net.Listener{port: ln},
	}
	s.startListeners()

	res := make(chan string)
	go func() { res <- get(t, "http://"+ln.Addr().String()+"/") }()
	<-entered

	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.shutdownServers(ctx)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("shutdown returned while a request was in flight")
	case <-time.After(200 * time.Millisecond):
	}

	close(release)
	assert.Equal(t, "done", <-res)
	<-done
}
