package msgpack

import (
	"github.com/vmihailenco/msgpack"

	"github.com/admpub/cache/encoding"
)

var MsgPack encoding.Codec = &msgPack{}

type msgPack struct {
}

func (_ *msgPack) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (_ *msgPack) Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}
