package utils_test

import (
	"os"
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetEnvEmptyFallback(t *testing.T) {
	os.Setenv("testing", "1")

	value := utils.GetEnv("testing", "")

	assert.Equal(t, "1", value)

	tearDown()
}

func TestGetEnvFilledFallback(t *testing.T) {
	os.Setenv("testing", "2")

	value := utils.GetEnv("testing", "3")

	assert.Equal(t, "2", value)

	tearDown()
}

func TestGetEnvMissingEnv(t *testing.T) {
	value := utils.GetEnv("testing", "1")

	assert.Equal(t, "1", value)

	tearDown()
}

func TestGetProxyURLWhenSet(t *testing.T) {
	os.Setenv("FORWARD_TO", "https://www.example.com")
	value := utils.GetProxyURL()

	assert.Equal(t, "https://www.example.com", value)

	tearDown()
}

func TestGetProxyURLWhenNotSet(t *testing.T) {
	value := utils.GetProxyURL()

	assert.Equal(t, "", value)
}

func tearDown() {
	os.Unsetenv("testing")
	os.Unsetenv("FORWARD_TO")
}
