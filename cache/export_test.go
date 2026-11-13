//go:build all || unit
// +build all unit

package cache

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import "net/url"

// MetadataKeyForTest - Exposes metadataKey to the external test package, which
// is where the rest of the key tests already live.
func MetadataKeyForTest(method string, target url.URL, variant string) string {
	return metadataKey(method, target, variant)
}
