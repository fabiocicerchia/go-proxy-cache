package utils_test

import (
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestGobEncodeDecode(t *testing.T) {
	str := []byte("test string")

	encoded := utils.EncodeGob(str)
	var decoded []byte
	utils.DecodeGob(encoded, &decoded)

	assert.Equal(t, str, decoded)
}
