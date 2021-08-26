package string_test

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

	utilsString "github.com/fabiocicerchia/go-proxy-cache/utils/string"
)

func TestNormalizeHTTP(t *testing.T) {
	assert.Equal(t, "http", utilsString.NormalizeScheme("http"))
	assert.Equal(t, "http", utilsString.NormalizeScheme("HTTP"))
	assert.Equal(t, "http", utilsString.NormalizeScheme("HttP"))
	assert.Equal(t, "http", utilsString.NormalizeScheme("HTtp"))
	assert.Equal(t, "http", utilsString.NormalizeScheme("HtTp"))
}

func TestNormalizeHTTPS(t *testing.T) {
	assert.Equal(t, "https", utilsString.NormalizeScheme("https"))
	assert.Equal(t, "https", utilsString.NormalizeScheme("HTTPS"))
	assert.Equal(t, "https", utilsString.NormalizeScheme("HttPs"))
	assert.Equal(t, "https", utilsString.NormalizeScheme("HTtps"))
	assert.Equal(t, "https", utilsString.NormalizeScheme("HtTpS"))
}

func TestNormalizeNonExisting(t *testing.T) {
	assert.Equal(t, "", utilsString.NormalizeScheme(""))
	assert.Equal(t, "", utilsString.NormalizeScheme("1"))
	assert.Equal(t, "", utilsString.NormalizeScheme("-"))
	assert.Equal(t, "", utilsString.NormalizeScheme("qwerty"))
	assert.Equal(t, "", utilsString.NormalizeScheme("wss"))
}
