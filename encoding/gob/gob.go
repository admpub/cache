package gob

import (
	"bytes"
	"encoding/gob"

	"github.com/admpub/cache/encoding"
)

var GOB encoding.Codec = &gobx{}

type gobx struct {
}

func (_ *gobx) Marshal(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := gob.NewEncoder(buf).Encode(v)
	return buf.Bytes(), err
}

func (_ *gobx) Unmarshal(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	return gob.NewDecoder(buf).Decode(v)
}
