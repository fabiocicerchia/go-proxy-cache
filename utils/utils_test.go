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

// --- CastToString

func TestCastToStringEmpty(t *testing.T) {
	input := []string{}
	value := utils.CastToString(input)

	assert.Equal(t, "", value)

	tearDown()
}

func TestCastToStringOneElement(t *testing.T) {
	input := []string{"a"}
	value := utils.CastToString(input)

	assert.Equal(t, "a", value)

	tearDown()
}

func TestCastToStringTwoElements(t *testing.T) {
	input := []string{"a", "b"}
	value := utils.CastToString(input)

	assert.Equal(t, "a", value)

	tearDown()
}

func tearDown() {
	config.Config = config.Configuration{}
	os.Unsetenv("testing")
}
