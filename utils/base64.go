package utils

import (
	"encoding/base64"
)

func Base64Encode(source []byte) []byte {
	buf := make([]byte, base64.URLEncoding.EncodedLen(len(source)))
	base64.URLEncoding.Encode(buf, source)

	return buf
}

func Base64Decode(source []byte) ([]byte, error) {
	dbuf := make([]byte, base64.URLEncoding.DecodedLen(len(source)))
	decodedBytes, err := base64.URLEncoding.Decode(dbuf, source)
	if err != nil {
		return nil, err
	}

	return dbuf[:decodedBytes], nil
}
