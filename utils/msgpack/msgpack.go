package msgpack

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"bytes"

	"github.com/ugorji/go/codec"
)

var msgpackHandler codec.MsgpackHandle

// Encode - Encodes object with msgpack.
func Encode(obj interface{}) ([]byte, error) {
	buff := new(bytes.Buffer)
	encoder := codec.NewEncoder(buff, &msgpackHandler)
	err := encoder.Encode(obj)

	return buff.Bytes(), err
}

// Decode - Decodes object with msgpack.
func Decode(b []byte, v interface{}) error {
	decoder := codec.NewDecoderBytes(b, &msgpackHandler)

	return decoder.Decode(v)
}
