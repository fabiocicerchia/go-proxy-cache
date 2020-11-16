// +build unit

package roundrobin_test

import (
	"fmt"
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/server/balancer/roundrobin"
	"github.com/stretchr/testify/assert"
)

func TestPickEmpty(t *testing.T) {
	b := roundrobin.New([]interface{}{})

	value, err := b.Pick()

	assert.NotNil(t, err)
	assert.Equal(t, "*errors.errorString", fmt.Sprintf("%T", err))
	assert.Equal(t, err.Error(), "no item is available")

	assert.Nil(t, value)
}

func TestPickWithData(t *testing.T) {
	b := roundrobin.New([]interface{}{
		"item1",
		"item2",
		"item3",
	})

	value, err := b.Pick()

	assert.Nil(t, err)

	assert.NotNil(t, value)
	assert.NotEmpty(t, value)
	assert.Regexp(t, "^(item1|item2|item3)$", value)
}

func TestPickCorrectness(t *testing.T) {
	b := roundrobin.New([]interface{}{
		"item1",
		"item2",
		"item3",
	})

	// first round (shuffling)
	var value1, value2, value3, value4 interface{}
	value1, err := b.Pick()
	assert.Nil(t, err)

	// second round (sequential)
	switch value1 {
	case "item1":
		value2, err = b.Pick()
		assert.Nil(t, err)
		assert.Equal(t, "item2", value2)
	case "item2":
		value2, err = b.Pick()
		assert.Nil(t, err)
		assert.Equal(t, "item3", value2)
	case "item3":
		value2, err = b.Pick()
		assert.Nil(t, err)
		assert.Equal(t, "item1", value2)
	}

	// third round (sequential)
	switch value2 {
	case "item1":
		value3, err = b.Pick()
		assert.Nil(t, err)
		assert.Equal(t, "item2", value3)
	case "item2":
		value3, err = b.Pick()
		assert.Nil(t, err)
		assert.Equal(t, "item3", value3)
	case "item3":
		value3, err = b.Pick()
		assert.Nil(t, err)
		assert.Equal(t, "item1", value3)
	}

	// fourth round (sequential)
	switch value3 {
	case "item1":
		value4, err = b.Pick()
		assert.Nil(t, err)
		assert.Equal(t, "item2", value4)
	case "item2":
		value4, err = b.Pick()
		assert.Nil(t, err)
		assert.Equal(t, "item3", value4)
	case "item3":
		value4, err = b.Pick()
		assert.Nil(t, err)
		assert.Equal(t, "item1", value4)
	}
}
