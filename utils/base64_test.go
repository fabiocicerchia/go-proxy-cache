// +build unit

package utils_test

import (
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestBase64EncodeDecode(t *testing.T) {
	str := []byte("test string")

	encoded := utils.Base64Encode(str)
	decoded, err := utils.Base64Decode(encoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}

func TestBase64CorruptedDecode(t *testing.T) {
	str := []byte("test string")

	encoded := utils.Base64Encode(str)
	decoded, err := utils.Base64Decode(encoded)
	assert.Nil(t, err)

	assert.Equal(t, str, decoded)
}
