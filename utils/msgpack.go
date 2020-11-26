package utils

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"bytes"

	"github.com/ugorji/go/codec"
)

var msgpackHandler codec.MsgpackHandle

// MsgpackEncode - Encodes object with msgpack.
func MsgpackEncode(obj interface{}) ([]byte, error) {
	buff := new(bytes.Buffer)
	encoder := codec.NewEncoder(buff, &msgpackHandler)
	err := encoder.Encode(obj)

	return buff.Bytes(), err
}

// MsgpackDecode - Decodes object with msgpack.
func MsgpackDecode(b []byte, v interface{}) error {
	decoder := codec.NewDecoderBytes(b, &msgpackHandler)

	return decoder.Decode(v)
}
