package cache_test

import (
	"testing"

	"github.com/admpub/cache"
	"github.com/stretchr/testify/assert"
)

func TestMemory(t *testing.T) {
	c, err := cache.NewCacher("memory", cache.Options{Interval: 300})
	assert.Nil(t, err)
	defer c.Close()
	data := &[]*User{
		&User{Name: "A", Age: 6},
		&User{Name: "B", Age: 7},
		&User{Name: "C", Age: 8},
	}
	err = c.Put("testkey", data, 86400)
	assert.Nil(t, err)
	recv := &[]*User{}
	err = c.Get("testkey", recv)
	assert.Nil(t, err)
	assert.Equal(t, data, recv)

	wrap := &Wrap{
		K: `test`, V: 100, X: data,
	}
	err = c.Put("testkey2", wrap, 86400)
	assert.Nil(t, err)
	recv2 := &Wrap{
		X: &[]*User{},
	}
	err = c.Get("testkey2", recv2)
	assert.Nil(t, err)
	assert.Equal(t, wrap, recv2)

	wraps := []*Wrap{wrap}
	err = c.Put("testkey3", wraps, 86400)
	assert.Nil(t, err)
	recv3 := []*Wrap{
		&Wrap{X: &[]*User{}},
	}
	err = c.Get("testkey3", &recv3)
	assert.Nil(t, err)
	assert.Equal(t, wraps, recv3)

	// modify
	wrapCopy := *wrap
	wrap.K = `modify`

	recv2 = &Wrap{
		X: &[]*User{},
	}
	err = c.Get("testkey2", recv2)
	assert.Nil(t, err)
	// 应该还是原值
	assert.Equal(t, &wrapCopy, recv2)
	assert.Equal(t, `test`, wrapCopy.K)
}
