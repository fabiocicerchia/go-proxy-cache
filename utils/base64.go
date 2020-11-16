package utils

import (
	"encoding/base64"
)

func Base64Encode(source []byte) string {
	return base64.StdEncoding.EncodeToString(source)
}

func Base64Decode(source string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(source)
}
