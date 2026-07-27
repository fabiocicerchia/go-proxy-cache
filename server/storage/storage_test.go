//go:build all || unit
// +build all unit

package storage_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/server/storage"
)

// --- ApplyNegativeTTL

func TestApplyNegativeTTLWithOverride(t *testing.T) {
	value := storage.ApplyNegativeTTL(404, map[int]int{404: 30, 502: 10}, 3600*time.Second)

	assert.Equal(t, 30*time.Second, value)
}

func TestApplyNegativeTTLWithoutOverride(t *testing.T) {
	value := storage.ApplyNegativeTTL(200, map[int]int{404: 30}, 3600*time.Second)

	assert.Equal(t, 3600*time.Second, value)
}

func TestApplyNegativeTTLWithNilMap(t *testing.T) {
	value := storage.ApplyNegativeTTL(200, nil, 3600*time.Second)

	assert.Equal(t, 3600*time.Second, value)
}
