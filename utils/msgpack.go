package utils

import (
	"bytes"

	"github.com/ugorji/go/codec"
)

var msgpackHandler codec.MsgpackHandle

func MsgpackEncode(obj interface{}) ([]byte, error) {
	buff := new(bytes.Buffer)
	encoder := codec.NewEncoder(buff, &msgpackHandler)
	err := encoder.Encode(obj)
	if err != nil {
		return nil, err
	}

	b := buff.Bytes()

	return b, nil
}

func MsgpackDecode(b []byte, v interface{}) error {
	decoder := codec.NewDecoderBytes(b, &msgpackHandler)
	return decoder.Decode(v)
}
