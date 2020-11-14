package utils

import (
	"encoding/base64"
	"log"
)

func Base64Encode(source []byte) []byte {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(source)))
	base64.StdEncoding.Encode(buf, source)
	return buf
}

func Base64Decode(source []byte) []byte {
	dbuf := make([]byte, base64.URLEncoding.DecodedLen(len(source)))
	decodedBytes, err := base64.URLEncoding.Decode(dbuf, source)
	if err != nil {
		log.Fatal(err)
	}
	return dbuf[:decodedBytes]
}
