package server

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetListenAddressWithDefault(t *testing.T) {
	value := getListenAddress()

	assert.Equal(t, ":8080", value)
}

func TestGetListenAddressWithCustom(t *testing.T) {
	os.Setenv("SERVER_PORT", "8081")

	value := getListenAddress()

	assert.Equal(t, ":8081", value)
}
