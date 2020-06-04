package cache

import (
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/assert"

	"github.com/admpub/cache"
)

func TestCache(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	c := New()
	c.StartAndGC(cache.Options{
		Adapter:       `redis`,
		AdapterConfig: `network=tcp,addr=` + s.Addr() + `,password=,db=0,pool_size=100,idle_timeout=180,hset_name=Cache,prefix=cache:`,
	})

	assert.Implements(t, (*cache.Cache)(nil), c)
	err = c.Put("exists", "exists", 86400)
	if assert.NoError(t, err) {
		var value string
		err = c.Get("exists", &value)
		assert.NoError(t, err)
		assert.Equal(t, "exists", value)
		assert.Equal(t, "exists", c.Any("exists"))
	} else {
		panic(err)
	}
	type dataBean struct {
		Name string
	}
	data := &dataBean{Name: "Cache"}
	err = c.Put("data", data, 86400)
	if assert.NoError(t, err) {
		value := &dataBean{}
		err = c.Get("data", &value)
		assert.NoError(t, err)
		assert.Equal(t, data, value)
	} else {
		panic(err)
	}
	var value string
	err = c.Get("non-exists", &value)
	assert.Equal(t, cache.ErrNotFound, err)
}
