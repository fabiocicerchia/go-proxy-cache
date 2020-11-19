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
