package msgpack

import (
	"github.com/shamaton/msgpack/v2"

	"github.com/admpub/cache/encoding"
)

// MsgPack default MsgPack codec
var MsgPack encoding.Codec = &msgPack{}

type msgPack struct {
}

func (_ *msgPack) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (_ *msgPack) Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}
