package scheme_test

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

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/utils/scheme"
)

func TestNormalizeHTTP(t *testing.T) {
	assert.Equal(t, "http", scheme.NormalizeScheme("http"))
	assert.Equal(t, "http", scheme.NormalizeScheme("HTTP"))
	assert.Equal(t, "http", scheme.NormalizeScheme("HttP"))
	assert.Equal(t, "http", scheme.NormalizeScheme("HTtp"))
	assert.Equal(t, "http", scheme.NormalizeScheme("HtTp"))
}

func TestNormalizeHTTPS(t *testing.T) {
	assert.Equal(t, "https", scheme.NormalizeScheme("https"))
	assert.Equal(t, "https", scheme.NormalizeScheme("HTTPS"))
	assert.Equal(t, "https", scheme.NormalizeScheme("HttPs"))
	assert.Equal(t, "https", scheme.NormalizeScheme("HTtps"))
	assert.Equal(t, "https", scheme.NormalizeScheme("HtTpS"))
}

func TestNormalizeNonExisting(t *testing.T) {
	assert.Equal(t, "", scheme.NormalizeScheme(""))
	assert.Equal(t, "", scheme.NormalizeScheme("1"))
	assert.Equal(t, "", scheme.NormalizeScheme("-"))
	assert.Equal(t, "", scheme.NormalizeScheme("qwerty"))
	assert.Equal(t, "", scheme.NormalizeScheme("wss"))
}
