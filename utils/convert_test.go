// +build unit

package utils_test

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

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/stretchr/testify/assert"
)

// --- ConvertToDuration

func TestConvertToDurationEmpty(t *testing.T) {
	value := utils.ConvertToDuration("")

	assert.Equal(t, time.Duration(0), value)

	tearDown()
}

func TestConvertToDurationSeconds(t *testing.T) {
	value := utils.ConvertToDuration("10s")

	assert.Equal(t, time.Duration(10*time.Second), value)

	tearDown()
}

func TestConvertToDurationDifferentValues(t *testing.T) {
	value := utils.ConvertToDuration("10m")
	assert.Equal(t, time.Duration(10*time.Minute), value)

	value = utils.ConvertToDuration("10h")
	assert.Equal(t, time.Duration(10*time.Hour), value)

	tearDown()
}

// --- ConvertToInt

func TestConvertToIntEmpty(t *testing.T) {
	value := utils.ConvertToInt("")

	assert.Equal(t, 0, value)

	tearDown()
}

func TestConvertToIntInvalid(t *testing.T) {
	value := utils.ConvertToInt("A")

	assert.Equal(t, 0, value)

	tearDown()
}

func TestConvertToIntValid(t *testing.T) {
	value := utils.ConvertToInt("123")

	assert.Equal(t, 123, value)

	tearDown()
}

// --- ConvertToIntSlice

func TestConvertToIntSliceEmpty(t *testing.T) {
	value := utils.ConvertToIntSlice([]string{})
	assert.Equal(t, []int{}, value)

	value = utils.ConvertToIntSlice([]string{""})
	assert.Equal(t, []int{0}, value)

	tearDown()
}

func TestConvertToIntSliceInvalid(t *testing.T) {
	value := utils.ConvertToIntSlice([]string{"A"})

	assert.Equal(t, []int{0}, value)

	tearDown()
}

func TestConvertToIntSliceValid(t *testing.T) {
	value := utils.ConvertToIntSlice([]string{"123", "345"})

	assert.Equal(t, []int{123, 345}, value)

	tearDown()
}
