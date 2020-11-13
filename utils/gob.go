package utils

import (
	"bytes"
	"encoding/gob"
	"log"
)

func EncodeGob(source interface{}) []byte {
	var buff bytes.Buffer
	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(source)
	if err != nil {
		log.Fatal(err)
	}
	return buff.Bytes()
}

func DecodeGob(source []byte, destination interface{}) {
	decoder := gob.NewDecoder(bytes.NewBuffer(source))
	err := decoder.Decode(destination)
	if err != nil {
		log.Fatal(err)
	}
}
