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
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"log"
	"time"

	_ "github.com/lib/pq"

	"github.com/admpub/cache"
	"github.com/admpub/cache/encoding"
)

// PostgresCacher represents a postgres cache adapter implementation.
type PostgresCacher struct {
	cache.GetAs
	codec    encoding.Codec
	c        *sql.DB
	interval int
}

// New creates and returns a new postgres cacher.
func New() cache.Cache {
	c := &PostgresCacher{codec: cache.DefaultCodec}
	c.GetAs = cache.GetAs{Cache: c}
	return c
}

func (c *PostgresCacher) SetCodec(codec encoding.Codec) {
	c.codec = codec
}

func (c *PostgresCacher) Codec() encoding.Codec {
	return c.codec
}

func (c *PostgresCacher) md5(key string) string {
	m := md5.Sum([]byte(key))
	return hex.EncodeToString(m[:])
}

// Put puts value into cache with key and expire time.
// If expired is 0, it will be deleted by next GC operation.
func (c *PostgresCacher) Put(key string, val interface{}, expire int64) error {
	item := &cache.Item{Val: val}
	data, err := c.codec.Marshal(item)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	if c.IsExist(key) {
		_, err = c.c.Exec("UPDATE cache SET data=$1, created=$2, expire=$3 WHERE key=$4", data, now, expire, c.md5(key))
	} else {
		_, err = c.c.Exec("INSERT INTO cache(key,data,created,expire) VALUES($1,$2,$3,$4)", c.md5(key), data, now, expire)
	}
	return err
}

func (c *PostgresCacher) read(key string, value interface{}) (*cache.Item, error) {
	var (
		data    []byte
		created int64
		expire  int64
	)
	err := c.c.QueryRow("SELECT data,created,expire FROM cache WHERE key=$1", c.md5(key)).Scan(&data, &created, &expire)
	if err != nil {
		return nil, err
	}

	item := &cache.Item{Val: value}
	if err = c.codec.Unmarshal(data, item); err != nil {
		return nil, err
	}
	item.Created = created
	item.Expire = expire
	return item, nil
}

// Get gets cached value by given key.
func (c *PostgresCacher) Get(key string, value interface{}) error {
	item, err := c.read(key, value)
	if err != nil {
		return err
	}
	if item.Val == nil {
		return cache.ErrNotFound
	}

	if item.Expire > 0 &&
		(time.Now().Unix()-item.Created) >= item.Expire {
		c.Delete(key)
		return cache.ErrExpired
	}
	return nil
}

// Delete deletes cached value by given key.
func (c *PostgresCacher) Delete(key string) error {
	_, err := c.c.Exec("DELETE FROM cache WHERE key=$1", c.md5(key))
	return err
}

// Incr increases cached int-type value by given key as a counter.
func (c *PostgresCacher) Incr(key string) error {
	var i int64
	item, err := c.read(key, &i)
	if err != nil {
		return err
	}

	item.Val, err = cache.Incr(i)
	if err != nil {
		return err
	}

	return c.Put(key, item.Val, item.Expire)
}

// Decr cached int value.
func (c *PostgresCacher) Decr(key string) error {
	var i int64
	item, err := c.read(key, &i)
	if err != nil {
		return err
	}

	item.Val, err = cache.Decr(i)
	if err != nil {
		return err
	}

	return c.Put(key, item.Val, item.Expire)
}

// IsExist returns true if cached value exists.
func (c *PostgresCacher) IsExist(key string) bool {
	var data []byte
	err := c.c.QueryRow("SELECT data FROM cache WHERE key=$1", c.md5(key)).Scan(&data)
	if err != nil && err != sql.ErrNoRows {
		panic("cache/postgres: error checking existence: " + err.Error())
	}
	return err != sql.ErrNoRows
}

// Flush deletes all cached data.
func (c *PostgresCacher) Flush() error {
	_, err := c.c.Exec("DELETE FROM cache")
	return err
}

func (c *PostgresCacher) startGC() {
	if c.interval < 1 {
		return
	}

	if _, err := c.c.Exec("DELETE FROM cache WHERE EXTRACT(EPOCH FROM NOW()) - created >= expire"); err != nil {
		log.Printf("cache/postgres: error garbage collecting: %v", err)
	}

	time.AfterFunc(time.Duration(c.interval)*time.Second, func() { c.startGC() })
}

// StartAndGC starts GC routine based on config string settings.
func (c *PostgresCacher) StartAndGC(opt cache.Options) (err error) {
	c.interval = opt.Interval

	c.c, err = sql.Open("postgres", opt.AdapterConfig)
	if err != nil {
		return err
	}
	if err = c.c.Ping(); err != nil {
		return err
	}

	go c.startGC()
	return nil
}

func (c *PostgresCacher) Close() error {
	c.interval = 0
	if c.c == nil {
		return nil
	}
	return c.c.Close()
}

func (c *PostgresCacher) Client() interface{} {
	return c.c
}

func AsClient(client interface{}) *sql.DB {
	return client.(*sql.DB)
}

func init() {
	cache.Register("postgres", New())
}
