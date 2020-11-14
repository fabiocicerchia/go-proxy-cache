package utils

import (
	"bytes"

	"github.com/ugorji/go/codec"
)

func MsgpackEncode(obj interface{}) ([]byte, error) {
	var handler codec.MsgpackHandle
	buff := new(bytes.Buffer)
	encoder := codec.NewEncoder(buff, &handler)
	err := encoder.Encode(obj)
	if err != nil {
		return nil, err
	}

	b := buff.Bytes()

	return b, nil
}

func MsgpackDecode(b []byte, v interface{}) error {
	var handler codec.MsgpackHandle
	decoder := codec.NewDecoderBytes(b, &handler)
	err := decoder.Decode(v)
	if err != nil {
		return err
	}

	return nil
}
