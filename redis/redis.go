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
	"context"
	"fmt"
	"strings"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/webx-top/com"

	"github.com/admpub/cache"
	"github.com/admpub/cache/encoding"
	"github.com/admpub/ini"
)

// RedisCacher represents a redis cache adapter implementation.
type RedisCacher struct {
	cache.GetAs
	codec      encoding.Codec
	c          *redis.Client
	options    *redis.Options
	prefix     string
	hsetName   string
	occupyMode bool
}

func (c *RedisCacher) SetCodec(codec encoding.Codec) {
	c.codec = codec
}

func (c *RedisCacher) Codec() encoding.Codec {
	return c.codec
}

// Put puts value into cache with key and expire time.
// If expired is 0, it lives forever.
func (c *RedisCacher) Put(ctx context.Context, key string, val interface{}, expire int64) error {
	key = c.prefix + key
	value, err := c.codec.Marshal(val)
	if err != nil {
		return err
	}
	if err := c.c.Set(ctx, key, com.Bytes2str(value), time.Duration(expire)*time.Second).Err(); err != nil {
		return err
	}
	if c.occupyMode {
		return nil
	}
	return c.c.HSet(ctx, c.hsetName, key, "0").Err()
}

// Get gets cached value by given key.
func (c *RedisCacher) Get(ctx context.Context, key string, value interface{}) error {
	val, err := c.c.Get(ctx, c.prefix+key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return cache.ErrNotFound
		}
		return err
	}
	if len(val) == 0 {
		return cache.ErrNotFound
	}

	return c.codec.Unmarshal(val, value)
}

// Delete deletes cached value by given key.
func (c *RedisCacher) Delete(ctx context.Context, key string) error {
	key = c.prefix + key
	if err := c.c.Del(ctx, key).Err(); err != nil {
		return err
	}

	if c.occupyMode {
		return nil
	}
	return c.c.HDel(ctx, c.hsetName, key).Err()
}

// Incr increases cached int-type value by given key as a counter.
func (c *RedisCacher) Incr(ctx context.Context, key string) error {
	if exist, err := c.IsExist(ctx, key); !exist {
		if err != nil {
			return err
		}
		return fmt.Errorf("key '%s' not exist", key)
	}
	return c.c.Incr(ctx, c.prefix+key).Err()
}

// Decr decreases cached int-type value by given key as a counter.
func (c *RedisCacher) Decr(ctx context.Context, key string) error {
	if exist, err := c.IsExist(ctx, key); !exist {
		if err != nil {
			return err
		}
		return fmt.Errorf("key '%s' not exist", key)
	}
	return c.c.Decr(ctx, c.prefix+key).Err()
}

// IsExist returns true if cached value exists.
func (c *RedisCacher) IsExist(ctx context.Context, key string) (bool, error) {
	if c.c.Exists(ctx, c.prefix+key).Val() > 0 {
		return true, nil
	}

	if !c.occupyMode {
		c.c.HDel(ctx, c.hsetName, c.prefix+key)
	}
	return false, nil
}

// Flush deletes all cached data.
func (c *RedisCacher) Flush(ctx context.Context) error {
	if c.occupyMode {
		return c.c.FlushDB(ctx).Err()
	}

	keys, err := c.c.HKeys(ctx, c.hsetName).Result()
	if err != nil {
		return err
	}
	if err = c.c.Del(ctx, keys...).Err(); err != nil {
		return err
	}
	return c.c.Del(ctx, c.hsetName).Err()
}

// StartAndGC starts GC routine based on config string settings.
// AdapterConfig: network=tcp,addr=:6379,password=123456,db=0,pool_size=100,idle_timeout=180,hset_name=Cache,prefix=cache:
func (c *RedisCacher) StartAndGC(ctx context.Context, opts cache.Options) error {
	c.hsetName = "Cache"
	c.occupyMode = opts.OccupyMode

	cfg, err := ini.Load([]byte(strings.Replace(opts.AdapterConfig, ",", "\n", -1)))
	if err != nil {
		return err
	}

	c.options = &redis.Options{
		Network: "tcp",
	}
	for k, v := range cfg.Section("").KeysHash() {
		switch k {
		case "network":
			c.options.Network = v
		case "addr":
			c.options.Addr = v
		case "password":
			c.options.Password = v
		case "db":
			c.options.DB = com.Int(v)
		case "pool_size":
			c.options.PoolSize = com.Int(v)
		case "idle_timeout":
			c.options.IdleTimeout, err = time.ParseDuration(v + "s")
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

	c.c = redis.NewClient(c.options)
	if err = c.c.Ping(ctx).Err(); err != nil {
		return err
	}

	return nil
}

func (c *RedisCacher) Close() error {
	if c.c == nil {
		return nil
	}
	return c.c.Close()
}

func (c *RedisCacher) Client() interface{} {
	return c.c
}

func (c *RedisCacher) Options() *redis.Options {
	return c.options
}

func (c *RedisCacher) Name() string {
	return cacheEngineRedis
}

const cacheEngineRedis = `redis`

func AsClient(client interface{}) *redis.Client {
	return client.(*redis.Client)
}

func New() cache.Cache {
	c := &RedisCacher{codec: cache.DefaultCodec}
	c.GetAs = cache.GetAs{Cache: c}
	return c
}

func init() {
	cache.Register(cacheEngineRedis, New())
}
