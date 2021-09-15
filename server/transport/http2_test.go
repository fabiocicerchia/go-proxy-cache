//go:build all || unit
// +build all unit

package transport_test

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

	"github.com/fabiocicerchia/go-proxy-cache/server/transport"
)

func TestParseSimple(t *testing.T) {
	links := transport.Parse("</style.css>; as=style; rel=preload")

	assert.Len(t, links, 1)

	assert.Equal(t, "/style.css", links[0].URL)
	assert.Equal(t, "preload", links[0].Rel)
	assert.Equal(t, "style", links[0].Params["as"])
}

func TestParseSimpleWithoutPreload(t *testing.T) {
	links := transport.Parse("</style.css>; as=style")

	assert.Len(t, links, 1)

	assert.Equal(t, "/style.css", links[0].URL)
	assert.Equal(t, "", links[0].Rel)
	assert.Equal(t, "style", links[0].Params["as"])
}

func TestParseTwoURL(t *testing.T) {
	links := transport.Parse("</style.css>; as=style; rel=preload, </favicon.ico>; as=image; rel=preload")

	assert.Len(t, links, 2)

	assert.Equal(t, "/style.css", links[0].URL)
	assert.Equal(t, "preload", links[0].Rel)
	assert.Equal(t, "style", links[0].Params["as"])

	assert.Equal(t, "/favicon.ico", links[1].URL)
	assert.Equal(t, "preload", links[1].Rel)
	assert.Equal(t, "image", links[1].Params["as"])
}

func TestParseWithNoPush(t *testing.T) {
	links := transport.Parse("</nginx.png>; as=image; rel=preload; nopush")

	assert.Len(t, links, 1)

	assert.Equal(t, "/nginx.png", links[0].URL)
	assert.Equal(t, "preload", links[0].Rel)
	assert.Equal(t, "image", links[0].Params["as"])
}

func TestParseMultiple(t *testing.T) {
	headers := []string{
		"</nginx.png>; as=image; rel=preload; nopush",
		"</style.css>; as=style; rel=preload, </favicon.ico>; as=image; rel=preload",
	}
	links := transport.ParseMultiple(headers)

	assert.Len(t, links, 3)

	assert.Equal(t, "/nginx.png", links[0].URL)
	assert.Equal(t, "preload", links[0].Rel)
	assert.Equal(t, "image", links[0].Params["as"])

	assert.Equal(t, "/style.css", links[1].URL)
	assert.Equal(t, "preload", links[1].Rel)
	assert.Equal(t, "style", links[1].Params["as"])

	assert.Equal(t, "/favicon.ico", links[2].URL)
	assert.Equal(t, "preload", links[2].Rel)
	assert.Equal(t, "image", links[2].Params["as"])
}
