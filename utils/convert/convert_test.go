//go:build all || unit
// +build all unit

package convert_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/utils/convert"
)

// --- ToDuration

func TestToDurationEmpty(t *testing.T) {
	value := convert.ToDuration("")

	assert.Equal(t, time.Duration(0), value)
}

func TestToDurationSeconds(t *testing.T) {
	value := convert.ToDuration("10s")

	assert.Equal(t, 10*time.Second, value)
}

func TestToDurationDifferentValues(t *testing.T) {
	value := convert.ToDuration("10m")
	assert.Equal(t, 10*time.Minute, value)

	value = convert.ToDuration("10h")
	assert.Equal(t, 10*time.Hour, value)
}

// --- ToInt

func TestToIntEmpty(t *testing.T) {
	value := convert.ToInt("")

	assert.Equal(t, 0, value)
}

func TestToIntInvalid(t *testing.T) {
	value := convert.ToInt("A")

	assert.Equal(t, 0, value)
}

func TestToIntValid(t *testing.T) {
	value := convert.ToInt("123")

	assert.Equal(t, 123, value)
}

// --- ToIntSlice

func TestToIntSliceEmpty(t *testing.T) {
	value := convert.ToIntSlice([]string{})
	assert.Equal(t, []int{}, value)

	value = convert.ToIntSlice([]string{""})
	assert.Equal(t, []int{0}, value)
}

func TestToIntSliceInvalid(t *testing.T) {
	value := convert.ToIntSlice([]string{"A"})

	assert.Equal(t, []int{0}, value)
}

func TestToIntSliceValid(t *testing.T) {
	value := convert.ToIntSlice([]string{"123", "345"})

	assert.Equal(t, []int{123, 345}, value)
}
