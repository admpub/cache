// Copyright 2018 The go-cache Authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package cache

import (
	"errors"
	"fmt"
	"os"

	"github.com/lunny/nodb"
	"github.com/lunny/nodb/config"

	"github.com/admpub/cache"
	"github.com/admpub/cache/encoding"
)

var (
	ErrDBExists = errors.New("database already exists")
)

// NodbCacher represents a nodb cache adapter implementation.
type NodbCacher struct {
	cache.GetAs
	codec    encoding.Codec
	dbs      *nodb.Nodb
	db       *nodb.DB
	filepath string
}

func (c *NodbCacher) SetCodec(codec encoding.Codec) {
	c.codec = codec
}

func (c *NodbCacher) Codec() encoding.Codec {
	return c.codec
}

// Put puts value into cache with key and expire time.
// If expired is 0, it lives forever.
func (c *NodbCacher) Put(key string, val interface{}, expire int64) (err error) {
	value, err := c.codec.Marshal(val)
	if err != nil {
		return err
	}
	kBytes := []byte(key)
	if err = c.db.Set(kBytes, value); err != nil {
		return err
	}

	if expire > 0 {
		_, err = c.db.Expire(kBytes, expire)
		return err
	}
	return nil
}

// Get gets cached value by given key.
func (c *NodbCacher) Get(key string, value interface{}) error {
	val, err := c.db.Get([]byte(key))
	if err != nil {
		return err
	}
	if len(val) == 0 {
		return cache.ErrNotFound
	}
	return c.codec.Unmarshal(val, value)
}

// Delete deletes cached value by given key.
func (c *NodbCacher) Delete(key string) error {
	_, err := c.db.Del([]byte(key))
	return err
}

// Incr increases cached int-type value by given key as a counter.
func (c *NodbCacher) Incr(key string) error {
	if !c.IsExist(key) {
		return fmt.Errorf("key '%s' not exist", key)
	}
	_, err := c.db.Incr([]byte(key))
	return err
}

// Decr decreases cached int-type value by given key as a counter.
func (c *NodbCacher) Decr(key string) error {
	if !c.IsExist(key) {
		return fmt.Errorf("key '%s' not exist", key)
	}
	_, err := c.db.Decr([]byte(key))
	return err
}

// IsExist returns true if cached value exists.
func (c *NodbCacher) IsExist(key string) bool {
	num, err := c.db.Exists([]byte(key))
	return err == nil && num > 0
}

func (c *NodbCacher) new() (err error) {
	if c.db != nil {
		return ErrDBExists
	}

	cfg := new(config.Config)
	cfg.DataDir = c.filepath
	c.dbs, err = nodb.Open(cfg)
	if err != nil {
		return fmt.Errorf("cache/nodb: error opening db: %v", err)
	}

	c.db, err = c.dbs.Select(0)
	return err
}

// Flush deletes all cached data.
func (c *NodbCacher) Flush() (err error) {
	if err = os.RemoveAll(c.filepath); err != nil {
		return err
	}

	c.dbs.Close()
	c.db = nil
	c.dbs = nil

	return c.new()
}

// StartAndGC starts GC routine based on config string settings.
func (c *NodbCacher) StartAndGC(opt cache.Options) error {
	c.filepath = opt.AdapterConfig
	return c.new()
}

func (c *NodbCacher) Close() error {
	if c.dbs == nil {
		return nil
	}
	c.dbs.Close()
	c.db = nil
	c.dbs = nil
	return nil
}

func (c *NodbCacher) Client() interface{} {
	return c.dbs
}

func AsClient(client interface{}) *nodb.Nodb {
	return client.(*nodb.Nodb)
}

func New() cache.Cache {
	c := &NodbCacher{codec: cache.DefaultCodec}
	c.GetAs = cache.GetAs{Cache: c}
	return c
}

func init() {
	cache.Register("nodb", New())
}
