//go:build all || unit
// +build all unit

package client

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
	"sync"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// Concurrent requests take the per-key lock in parallel; this used to crash
// the process with "fatal error: concurrent map writes". No Redis is needed:
// the lock simply fails against an unreachable address.
func TestLockUnlockConcurrentKeysDoesNotRace(t *testing.T) {
	logger, _ := test.NewNullLogger()
	rdb := Connect("testing", config.Cache{Hosts: []string{"127.0.0.1:1"}}, logger)
	defer rdb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail fast instead of retrying against the unreachable Redis

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i)
			mutex, err := rdb.lock(ctx, key)
			assert.Error(t, err)
			assert.Nil(t, mutex)
			assert.Error(t, rdb.unlock(ctx, key, rdb.Redsync.NewMutex("mutex-testing-"+key)))
		}(i)
	}
	wg.Wait()
}
