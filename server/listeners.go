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
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

// listenFDsStart - First descriptor passed by systemd socket activation.
const listenFDsStart = 3

// envReexecFDs - Listener descriptors carried across a SIGHUP re-exec. They keep
// their numbers: moving them to 3.. would clobber fds the running process still uses.
const envReexecFDs = "GO_PROXY_CACHE_LISTEN_FDS"

// inheritedListeners - Returns the listening sockets handed to the process, by port.
func inheritedListeners() (map[string]net.Listener, error) {
	fds, err := inheritedFDs()
	if err != nil {
		return nil, err
	}

	listeners := make(map[string]net.Listener, len(fds))

	for _, fd := range fds {
		f := os.NewFile(uintptr(fd), "listener")
		ln, err := net.FileListener(f) // dups fd, so f can go
		_ = f.Close()

		if err != nil {
			return nil, fmt.Errorf("inherited fd %d is not a listening socket: %w", fd, err)
		}

		addr, ok := ln.Addr().(*net.TCPAddr)
		if !ok {
			_ = ln.Close()
			return nil, fmt.Errorf("inherited fd %d is not a TCP listener", fd)
		}

		listeners[strconv.Itoa(addr.Port)] = ln
	}

	return listeners, nil
}

// inheritedFDs - Reads (and clears, so nothing spawned later sees them) the
// re-exec variable or systemd's LISTEN_FDS/LISTEN_PID.
func inheritedFDs() ([]int, error) {
	if v, ok := os.LookupEnv(envReexecFDs); ok {
		_ = os.Unsetenv(envReexecFDs)

		var fds []int
		for _, s := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' }) {
			fd, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("%s=%q: %w", envReexecFDs, v, err)
			}
			fds = append(fds, fd)
		}

		return fds, nil
	}

	n, pid := os.Getenv("LISTEN_FDS"), os.Getenv("LISTEN_PID")
	_ = os.Unsetenv("LISTEN_FDS")
	_ = os.Unsetenv("LISTEN_PID")
	_ = os.Unsetenv("LISTEN_FDNAMES")

	count, err := strconv.Atoi(n)
	if err != nil || count <= 0 {
		return nil, nil //nolint:nilerr // not socket-activated
	}

	// LISTEN_PID is optional here so a supervisor that cannot know the pid
	// before exec (a shell, exec.Cmd) can still pass sockets.
	if pid != "" && pid != strconv.Itoa(os.Getpid()) {
		return nil, nil
	}

	fds := make([]int, count)
	for i := range fds {
		fds[i] = listenFDsStart + i
	}

	return fds, nil
}

// listenerFiles - Duplicates every listener's socket, so it outlives Shutdown.
func (s *Servers) listenerFiles() ([]*os.File, error) {
	var files []*os.File

	for _, group := range []map[string]*Server{s.HTTP, s.HTTPS} {
		for port, srv := range group {
			ln, ok := srv.Listener.(*net.TCPListener)
			if !ok {
				return nil, fmt.Errorf("listener on port %s is not TCP", port)
			}

			f, err := ln.File()
			if err != nil {
				return nil, fmt.Errorf("listener on port %s: %w", port, err)
			}

			files = append(files, f)
		}
	}

	return files, nil
}

// reexec - Replaces the process image with the current binary (same pid, same
// arguments), handing it the listening sockets. Returns only on failure.
func reexec(files []*os.File) error {
	fds := make([]string, len(files))

	for i, f := range files {
		if _, err := unix.FcntlInt(f.Fd(), unix.F_SETFD, 0); err != nil { // clear close-on-exec
			return err
		}

		fds[i] = strconv.Itoa(int(f.Fd()))
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	err = syscall.Exec(exe, os.Args, append(os.Environ(), envReexecFDs+"="+strings.Join(fds, ",")))
	runtime.KeepAlive(files) // a finalizer must not close the fds before exec

	return err
}

// connStates - The state of every open connection, so a drain knows when the
// requests it accepted are done.
type connStates struct {
	mu sync.Mutex
	m  map[net.Conn]http.ConnState
}

func (c *connStates) track(conn net.Conn, state http.ConnState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.m == nil {
		c.m = make(map[net.Conn]http.ConnState)
	}

	if state == http.StateClosed || state == http.StateHijacked {
		delete(c.m, conn)
	} else {
		c.m[conn] = state
	}
}

// busy - Whether any connection is waiting for, or serving, a request.
func (c *connStates) busy() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, state := range c.m {
		if state != http.StateIdle {
			return true
		}
	}

	return false
}
