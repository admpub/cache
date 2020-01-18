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
	"strings"
	"time"

	"github.com/webx-top/com"
	"gopkg.in/redis.v2"

	"github.com/admpub/cache"
	"github.com/admpub/cache/encoding"
	"github.com/admpub/ini"
)

// RedisCacher represents a redis cache adapter implementation.
type RedisCacher struct {
	cache.GetAs
	codec      encoding.Codec
	c          *redis.Client
	prefix     string
	hsetName   string
	occupyMode bool
}

func (c *RedisCacher) SetCodec(codec encoding.Codec) {
	c.codec = codec
}

// Put puts value into cache with key and expire time.
// If expired is 0, it lives forever.
func (c *RedisCacher) Put(key string, val interface{}, expire int64) error {
	key = c.prefix + key
	value, err := c.codec.Marshal(val)
	if err != nil {
		return err
	}
	if expire == 0 {
		if err := c.c.Set(key, com.Bytes2str(value)).Err(); err != nil {
			return err
		}
	} else {
		if err := c.c.SetEx(key, time.Duration(expire)*time.Second, com.Bytes2str(value)).Err(); err != nil {
			return err
		}
	}

	if c.occupyMode {
		return nil
	}
	return c.c.HSet(c.hsetName, key, "0").Err()
}

// Get gets cached value by given key.
func (c *RedisCacher) Get(key string, value interface{}) error {
	val, err := c.c.Get(c.prefix + key).Result()
	if err != nil {
		if err == redis.Nil {
			return cache.ErrNotFound
		}
		return err
	}
	if len(val) == 0 {
		return cache.ErrNotFound
	}

	return c.codec.Unmarshal(com.Str2bytes(val), value)
}

// Delete deletes cached value by given key.
func (c *RedisCacher) Delete(key string) error {
	key = c.prefix + key
	if err := c.c.Del(key).Err(); err != nil {
		return err
	}

	if c.occupyMode {
		return nil
	}
	return c.c.HDel(c.hsetName, key).Err()
}

// Incr increases cached int-type value by given key as a counter.
func (c *RedisCacher) Incr(key string) error {
	if !c.IsExist(key) {
		return fmt.Errorf("key '%s' not exist", key)
	}
	return c.c.Incr(c.prefix + key).Err()
}

// Decr decreases cached int-type value by given key as a counter.
func (c *RedisCacher) Decr(key string) error {
	if !c.IsExist(key) {
		return fmt.Errorf("key '%s' not exist", key)
	}
	return c.c.Decr(c.prefix + key).Err()
}

// IsExist returns true if cached value exists.
func (c *RedisCacher) IsExist(key string) bool {
	if c.c.Exists(c.prefix + key).Val() {
		return true
	}

	if !c.occupyMode {
		c.c.HDel(c.hsetName, c.prefix+key)
	}
	return false
}

// Flush deletes all cached data.
func (c *RedisCacher) Flush() error {
	if c.occupyMode {
		return c.c.FlushDb().Err()
	}

	keys, err := c.c.HKeys(c.hsetName).Result()
	if err != nil {
		return err
	}
	if err = c.c.Del(keys...).Err(); err != nil {
		return err
	}
	return c.c.Del(c.hsetName).Err()
}

// StartAndGC starts GC routine based on config string settings.
// AdapterConfig: network=tcp,addr=:6379,password=123456,db=0,pool_size=100,idle_timeout=180,hset_name=Cache,prefix=cache:
func (c *RedisCacher) StartAndGC(opts cache.Options) error {
	c.hsetName = "Cache"
	c.occupyMode = opts.OccupyMode

	cfg, err := ini.Load([]byte(strings.Replace(opts.AdapterConfig, ",", "\n", -1)))
	if err != nil {
		return err
	}

	opt := &redis.Options{
		Network: "tcp",
	}
	for k, v := range cfg.Section("").KeysHash() {
		switch k {
		case "network":
			opt.Network = v
		case "addr":
			opt.Addr = v
		case "password":
			opt.Password = v
		case "db":
			opt.DB = com.StrTo(v).MustInt64()
		case "pool_size":
			opt.PoolSize = com.StrTo(v).MustInt()
		case "idle_timeout":
			opt.IdleTimeout, err = time.ParseDuration(v + "s")
			if err != nil {
				return fmt.Errorf("error parsing idle timeout: %v", err)
			}
		case "hset_name":
			c.hsetName = v
		case "prefix":
			c.prefix = v
		default:
			return fmt.Errorf("cache/redis: unsupported option '%s'", k)
		}
	}

	c.c = redis.NewClient(opt)
	if err = c.c.Ping().Err(); err != nil {
		return err
	}

	return nil
}

func New() cache.Cache {
	c := &RedisCacher{codec: cache.DefaultCodec}
	c.GetAs = cache.GetAs{Cache: c}
	return c
}

func init() {
	cache.Register("redis", New())
}
