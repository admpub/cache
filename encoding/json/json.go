package json

import (
	"encoding/json"

	"github.com/admpub/cache/encoding"
)

var JSON encoding.Codec = &jsonx{}

type jsonx struct {
}

func (_ *jsonx) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (_ *jsonx) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
