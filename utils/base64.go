package utils

import (
	"encoding/base64"
	"log"
)

func Base64Encode(source []byte) []byte {
	encodedBytes := make([]byte, base64.URLEncoding.EncodedLen(len(source)))
	base64.URLEncoding.Encode(encodedBytes, source)

	return encodedBytes
}

func Base64Decode(source []byte) []byte {
	decodedBytes := make([]byte, base64.URLEncoding.DecodedLen(len(source)))
	_, err := base64.URLEncoding.Decode(decodedBytes, source)
	if err != nil {
		log.Fatal(err)
	}
	return decodedBytes
}
