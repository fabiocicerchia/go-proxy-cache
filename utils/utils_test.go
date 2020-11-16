// +build unit

package utils_test

import (
	"os"
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/stretchr/testify/assert"
)

// --- GetEnv

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

// --- IfEmpty

func TestIfEmptyWithValue(t *testing.T) {
	value := utils.IfEmpty("text", "fallback")

	assert.Equal(t, "text", value)

	tearDown()
}

func TestIfEmptyWithoutValue(t *testing.T) {
	value := utils.IfEmpty("", "fallback")

	assert.Equal(t, "fallback", value)

	tearDown()
}

// --- Contains

func TestContainsEmpty(t *testing.T) {
	match := utils.Contains([]string{}, "d")

	assert.False(t, match)

	tearDown()
}

func TestContainsNoValue(t *testing.T) {
	match := utils.Contains([]string{"a", "b", "c"}, "d")

	assert.False(t, match)

	tearDown()
}

func TestContainsValue(t *testing.T) {
	match := utils.Contains([]string{"a", "b", "c"}, "c")

	assert.True(t, match)

	tearDown()
}

// --- GetHeaders

func TestGetHeadersEmpty(t *testing.T) {
	var input map[string][]string
	headers := utils.GetHeaders(input)

	assert.Len(t, headers, 0)

	tearDown()
}

func TestGetHeadersNotEmpty(t *testing.T) {
	input := make(map[string][]string)
	input["key"] = []string{"a", "b", "c"}

	headers := utils.GetHeaders(input)

	assert.Len(t, headers, 1)
	assert.Equal(t, "a b c", headers["key"])

	tearDown()
}

func tearDown() {
	config.Config = config.Configuration{}
	os.Unsetenv("testing")
}
