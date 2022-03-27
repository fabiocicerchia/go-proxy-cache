package base64

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"encoding/base64"
)

// Encode - Encodes object with base64.
func Encode(source []byte) string {
	return base64.StdEncoding.EncodeToString(source)
}

// Decode - Decodes object with base64.
func Decode(source string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(source)
}
