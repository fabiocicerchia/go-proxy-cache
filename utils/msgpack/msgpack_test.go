// +build all unit

package msgpack_test

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

	"github.com/fabiocicerchia/go-proxy-cache/utils/msgpack"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	str := []byte("test string")

	encoded, err := msgpack.Encode(str)
	assert.Nil(t, err)

	var decoded []byte
	err = msgpack.Decode(encoded, &decoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}
