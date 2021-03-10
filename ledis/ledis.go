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
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ledisdb/ledisdb/config"
	"github.com/ledisdb/ledisdb/ledis"
	"github.com/webx-top/com"

	"github.com/admpub/cache"
	"github.com/admpub/cache/encoding"
	"github.com/admpub/ini"
)

var defaultHSetName = []byte("Cache")

// LedisCacher represents a ledis cache adapter implementation.
type LedisCacher struct {
	cache.GetAs
	codec    encoding.Codec
	c        *ledis.Ledis
	db       *ledis.DB
	interval int
}

func (c *LedisCacher) SetCodec(codec encoding.Codec) {
	c.codec = codec
}

func (c *LedisCacher) Codec() encoding.Codec {
	return c.codec
}

// Put puts value into cache with key and expire time.
// If expired is 0, it lives forever.
func (c *LedisCacher) Put(key string, val interface{}, expire int64) (err error) {
	value, err := c.codec.Marshal(val)
	if err != nil {
		return err
	}
	kBytes := []byte(key)
	if expire == 0 {
		if err = c.db.Set(kBytes, value); err != nil {
			return err
		}
		_, err = c.db.HSet(kBytes, defaultHSetName, []byte("0"))
		return err
	}

	if err = c.db.SetEX(kBytes, expire, value); err != nil {
		return err
	}
	_, err = c.db.HSet(kBytes, defaultHSetName, []byte(com.ToStr(time.Now().Add(time.Duration(expire)*time.Second).Unix())))
	return err
}

// Get gets cached value by given key.
func (c *LedisCacher) Get(key string, value interface{}) error {
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
func (c *LedisCacher) Delete(key string) (err error) {
	if _, err = c.db.Del([]byte(key)); err != nil {
		return err
	}
	_, err = c.db.HDel(defaultHSetName, []byte(key))
	return err
}

// Incr increases cached int-type value by given key as a counter.
func (c *LedisCacher) Incr(key string) error {
	if !c.IsExist(key) {
		return fmt.Errorf("key '%s' not exist", key)
	}
	_, err := c.db.Incr([]byte(key))
	return err
}

// Decr decreases cached int-type value by given key as a counter.
func (c *LedisCacher) Decr(key string) error {
	if !c.IsExist(key) {
		return fmt.Errorf("key '%s' not exist", key)
	}
	_, err := c.db.Decr([]byte(key))
	return err
}

// IsExist returns true if cached value exists.
func (c *LedisCacher) IsExist(key string) bool {
	count, err := c.db.Exists([]byte(key))
	if err == nil && count > 0 {
		return true
	}
	c.db.HDel(defaultHSetName, []byte(key))
	return false
}

// Flush deletes all cached data.
func (c *LedisCacher) Flush() error {
	// FIXME: there must be something wrong, shouldn't use this one.
	_, err := c.db.FlushAll()
	return err

	// keys, err := c.c.HKeys(defaultHSetName)
	// if err != nil {
	// 	return err
	// }
	// if _, err = c.c.Del(keys...); err != nil {
	// 	return err
	// }
	// _, err = c.c.Del(defaultHSetName)
	// return err
}

func (c *LedisCacher) startGC() {
	if c.interval < 1 {
		return
	}

	kvs, err := c.db.HGetAll(defaultHSetName)
	if err != nil {
		log.Printf("cache/ledis: error garbage collecting(get): %v", err)
		return
	}

	now := time.Now().Unix()
	for _, v := range kvs {
		expire := com.Int64(v.Value)
		if expire == 0 || now < expire {
			continue
		}

		if err = c.Delete(string(v.Field)); err != nil {
			log.Printf("cache/ledis: error garbage collecting(delete): %v", err)
			continue
		}
	}

	time.AfterFunc(time.Duration(c.interval)*time.Second, func() { c.startGC() })
}

// StartAndGC starts GC routine based on config string settings.
// AdapterConfig: data_dir=./app.db,db=0
func (c *LedisCacher) StartAndGC(opts cache.Options) error {
	c.interval = opts.Interval

	cfg, err := ini.Load([]byte(strings.Replace(opts.AdapterConfig, ",", "\n", -1)))
	if err != nil {
		return err
	}

	db := 0
	opt := new(config.Config)
	for k, v := range cfg.Section("").KeysHash() {
		switch k {
		case "data_dir":
			opt.DataDir = v
		case "db":
			db = com.Int(v)
		default:
			return fmt.Errorf("cache/ledis: unsupported option '%s'", k)
		}
	}

	c.c, err = ledis.Open(opt)
	if err != nil {
		return fmt.Errorf("cache/ledis: error opening db: %v", err)
	}
	c.db, err = c.c.Select(db)
	if err != nil {
		return err
	}

	go c.startGC()
	return nil
}

func (c *LedisCacher) Close() error {
	c.interval = 0
	if c.c == nil {
		return nil
	}
	c.c.Close()
	return nil
}

func (c *LedisCacher) Client() interface{} {
	return c.c
}

func AsClient(client interface{}) *ledis.Ledis {
	return client.(*ledis.Ledis)
}

func New() cache.Cache {
	c := &LedisCacher{codec: cache.DefaultCodec}
	c.GetAs = cache.GetAs{Cache: c}
	return c
}

func init() {
	cache.Register("ledis", New())
}
