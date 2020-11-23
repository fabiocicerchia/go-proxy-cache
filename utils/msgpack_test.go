//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache
// +build unit

package utils_test

import (
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestMsgpackEncodeDecode(t *testing.T) {
	str := []byte("test string")

	encoded, err := utils.MsgpackEncode(str)
	assert.Nil(t, err)

	var decoded []byte
	err = utils.MsgpackDecode(encoded, &decoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}
