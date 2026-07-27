//go:build all || unit
// +build all unit

package config_test

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

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// --- EffectiveAllowedStatuses

func TestEffectiveAllowedStatusesWithoutNegativeTTL(t *testing.T) {
	cache := config.Cache{
		AllowedStatuses: []int{200, 301},
	}

	assert.ElementsMatch(t, []int{200, 301}, cache.EffectiveAllowedStatuses())
}

func TestEffectiveAllowedStatusesMergesNegativeTTLStatuses(t *testing.T) {
	cache := config.Cache{
		AllowedStatuses: []int{200},
		NegativeTTL:     map[int]int{404: 30, 502: 10},
	}

	assert.ElementsMatch(t, []int{200, 404, 502}, cache.EffectiveAllowedStatuses())
}

func TestEffectiveAllowedStatusesDoesNotDuplicate(t *testing.T) {
	cache := config.Cache{
		AllowedStatuses: []int{200, 404},
		NegativeTTL:     map[int]int{404: 30},
	}

	assert.ElementsMatch(t, []int{200, 404}, cache.EffectiveAllowedStatuses())
}
