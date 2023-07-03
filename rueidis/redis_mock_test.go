package cache

import (
	"context"
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

	ctx := context.Background()
	c := New()
	err = c.StartAndGC(ctx, cache.Options{
		Adapter:       `redis`,
		AdapterConfig: `network=tcp,addr=` + s.Addr() + `,password=,db=0,pool_size=100,idle_timeout=180,hset_name=Cache,prefix=cache:`,
	})
	assert.NoError(t,err)

	assert.Implements(t, (*cache.Cache)(nil), c)
	err = c.Put(ctx, "exists", "exists", 86400)
	if assert.NoError(t, err) {
		var value string
		err = c.Get(ctx, "exists", &value)
		assert.NoError(t, err)
		assert.Equal(t, "exists", value)
		assert.Equal(t, "exists", c.Any(ctx, "exists"))
	} else {
		panic(err)
	}
	type dataBean struct {
		Name string
	}
	data := &dataBean{Name: "Cache"}
	err = c.Put(ctx, "data", data, 86400)
	if assert.NoError(t, err) {
		value := &dataBean{}
		err = c.Get(ctx, "data", &value)
		assert.NoError(t, err)
		assert.Equal(t, data, value)
	} else {
		panic(err)
	}
	var value string
	err = c.Get(ctx, "non-exists", &value)
	assert.Equal(t, cache.ErrNotFound, err)
	exist, err := c.IsExist(ctx, "non-exists")
	assert.NoError(t, err)
	assert.Equal(t, false, exist)
	exist, err = c.IsExist(ctx, "exists")
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}
