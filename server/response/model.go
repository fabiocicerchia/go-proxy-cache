package response

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

// DataChunks - Internal structure for chunks of bytes.
type DataChunks [][]byte

// Bytes - Returns flat slice of bytes.
func (dc DataChunks) Bytes() []byte {
	bytes := []byte{}

	for _, c := range dc {
		bytes = append(bytes, c...)
	}

	return bytes
}

// Len - Returns total length.
func (dc DataChunks) Len() int {
	return slice.LenSliceBytes(([][]byte)(dc))
}
