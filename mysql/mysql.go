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
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/admpub/cache"
	"github.com/admpub/cache/encoding"
)

// MysqlCacher represents a mysql cache adapter implementation.
type MysqlCacher struct {
	cache.GetAs
	codec    encoding.Codec
	c        *sql.DB
	interval int
}

// New creates and returns a new mysql cacher.
func New() cache.Cache {
	c := &MysqlCacher{codec: cache.DefaultCodec}
	c.GetAs = cache.GetAs{Cache: c}
	return c
}

func (c *MysqlCacher) SetCodec(codec encoding.Codec) {
	c.codec = codec
}

func (c *MysqlCacher) Codec() encoding.Codec {
	return c.codec
}

func (c *MysqlCacher) md5(key string) string {
	m := md5.Sum([]byte(key))
	return hex.EncodeToString(m[:])
}

// Put puts value into cache with key and expire time.
// If expired is 0, it will be deleted by next GC operation.
func (c *MysqlCacher) Put(ctx context.Context, key string, val interface{}, expire int64) error {
	item := cache.CacheItemPoolGet()
	item.Val = val
	data, err := c.codec.Marshal(item)
	cache.CacheItemPoolRelease(item)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	_, err = c.c.ExecContext(ctx, "REPLACE INTO cache(`key`,data,created,expire) VALUES(?,?,?,?)", c.md5(key), data, now, expire)
	return err
}

func (c *MysqlCacher) read(ctx context.Context, key string, value interface{}) (*cache.Item, error) {
	var (
		data    []byte
		created int64
		expire  int64
	)
	err := c.c.QueryRowContext(ctx, "SELECT data,created,expire FROM cache WHERE `key`=?", c.md5(key)).Scan(&data, &created, &expire)
	if err != nil {
		return nil, err
	}

	item := cache.CacheItemPoolGet()
	item.Val = value
	if err = c.codec.Unmarshal(data, item); err != nil {
		return nil, err
	}
	item.Created = created
	item.Expire = expire
	return item, nil
}

// Get gets cached value by given key.
func (c *MysqlCacher) Get(ctx context.Context, key string, value interface{}) error {
	item, err := c.read(ctx, key, value)
	if item != nil {
		defer cache.CacheItemPoolRelease(item)
	}
	if err != nil {
		return nil
	}
	if item.Val == nil {
		return cache.ErrNotFound
	}

	if item.Expire > 0 &&
		(time.Now().Unix()-item.Created) >= item.Expire {
		c.Delete(ctx, key)
		return cache.ErrExpired
	}
	return nil
}

// Delete deletes cached value by given key.
func (c *MysqlCacher) Delete(ctx context.Context, key string) error {
	_, err := c.c.ExecContext(ctx, "DELETE FROM cache WHERE `key`=?", c.md5(key))
	return err
}

// Incr increases cached int-type value by given key as a counter.
func (c *MysqlCacher) Incr(ctx context.Context, key string) error {
	var i int64
	item, err := c.read(ctx, key, &i)
	if item != nil {
		defer cache.CacheItemPoolRelease(item)
	}
	if err != nil {
		return err
	}

	item.Val, err = cache.Incr(i)
	if err != nil {
		return err
	}

	return c.Put(ctx, key, item.Val, item.Expire)
}

// Decr cached int value.
func (c *MysqlCacher) Decr(ctx context.Context, key string) error {
	var i int64
	item, err := c.read(ctx, key, i)
	if item != nil {
		defer cache.CacheItemPoolRelease(item)
	}
	if err != nil {
		return err
	}

	item.Val, err = cache.Decr(item.Val)
	if err != nil {
		return err
	}

	return c.Put(ctx, key, item.Val, item.Expire)
}

// IsExist returns true if cached value exists.
func (c *MysqlCacher) IsExist(ctx context.Context, key string) (bool, error) {
	var data []byte
	err := c.c.QueryRowContext(ctx, "SELECT data FROM cache WHERE `key`=?", c.md5(key)).Scan(&data)
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("cache/mysql: error checking existence: %w", err)
		return false, err
	}
	return err != sql.ErrNoRows, nil
}

// Flush deletes all cached data.
func (c *MysqlCacher) Flush(ctx context.Context) error {
	_, err := c.c.ExecContext(ctx, "DELETE FROM cache")
	return err
}

func (c *MysqlCacher) startGC(ctx context.Context) {
	if c.interval < 1 {
		return
	}

	if _, err := c.c.ExecContext(ctx, "DELETE FROM cache WHERE UNIX_TIMESTAMP(NOW()) - created >= expire"); err != nil {
		log.Printf("cache/mysql: error garbage collecting: %v", err)
	}

	time.AfterFunc(time.Duration(c.interval)*time.Second, func() { c.startGC(ctx) })
}

// StartAndGC starts GC routine based on config string settings.
func (c *MysqlCacher) StartAndGC(ctx context.Context, opt cache.Options) (err error) {
	c.interval = opt.Interval

	c.c, err = sql.Open("mysql", opt.AdapterConfig)
	if err != nil {
		return err
	}
	if err = c.c.Ping(); err != nil {
		return err
	}

	if _, err = c.c.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS cache ("+
		"	`key` char(32) NOT NULL,"+
		"	`data` longblob NOT NULL,"+
		"	`created` int(11) unsigned NOT NULL DEFAULT '0',"+
		"	`expire` int(11) unsigned NOT NULL DEFAULT '0',"+
		"	PRIMARY KEY (`key`)"+
		"  ) ENGINE=InnoDB;"); err != nil {
		return err
	}

	go c.startGC(ctx)
	return nil
}

func (c *MysqlCacher) Close() error {
	c.interval = 0
	if c.c == nil {
		return nil
	}
	return c.c.Close()
}

func (c *MysqlCacher) Client() interface{} {
	return c.c
}

func (c *MysqlCacher) Name() string {
	return cacheEngineMysql
}

const cacheEngineMysql = `mysql`

func AsClient(client interface{}) *sql.DB {
	return client.(*sql.DB)
}

func init() {
	cache.Register(cacheEngineMysql, New())
}
