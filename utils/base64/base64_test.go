//go:build all || unit
// +build all unit

package base64_test

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

	"github.com/fabiocicerchia/go-proxy-cache/utils/base64"
)

func TestEncodeDecode(t *testing.T) {
	str := []byte("test string")

	encoded := base64.Encode(str)
	decoded, err := base64.Decode(encoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}

func TestBase64CorruptedDecode(t *testing.T) {
	str := []byte("test string")

	encoded := base64.Encode(str)
	decoded, err := base64.Decode(encoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}
