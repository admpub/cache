package sqlite

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/admpub/cache"
	"github.com/admpub/cache/encoding"
	"github.com/admpub/cove"
	_ "github.com/admpub/cove/driver"
)

// SQLiteCacher represents a SQLite cache adapter implementation.
type SQLiteCacher struct {
	cache.GetAs
	codec    encoding.Codec
	c        *cove.Cache
	interval int
}

// New creates and returns a new SQLite cacher.
func New() cache.Cache {
	c := &SQLiteCacher{codec: cache.DefaultCodec}
	c.GetAs = cache.GetAs{Cache: c}
	return c
}

func (c *SQLiteCacher) SetCodec(codec encoding.Codec) {
	c.codec = codec
}

func (c *SQLiteCacher) Codec() encoding.Codec {
	return c.codec
}

func (c *SQLiteCacher) md5(key string) string {
	m := md5.Sum([]byte(key))
	return hex.EncodeToString(m[:])
}

// Put puts value into cache with key and expire time.
// If expired is 0, it will be deleted by next GC operation.
func (c *SQLiteCacher) Put(ctx context.Context, key string, val interface{}, expire int64) error {
	data, err := c.codec.Marshal(val)
	if err != nil {
		return err
	}
	if expire <= 0 {
		return c.c.Set(c.md5(key), data)
	}
	return c.c.SetTTL(c.md5(key), data, time.Second*time.Duration(expire))
}

func (c *SQLiteCacher) read(key string, value interface{}) error {
	data, err := c.c.Get(c.md5(key))
	if err != nil {
		return err
	}

	return c.codec.Unmarshal(data, value)
}

// Get gets cached value by given key.
func (c *SQLiteCacher) Get(ctx context.Context, key string, value interface{}) error {
	err := c.read(key, value)
	if err != nil {
		if errors.Is(err, cove.NotFound) {
			return cache.ErrNotFound
		}
	}
	return err
}

// Delete deletes cached value by given key.
func (c *SQLiteCacher) Delete(ctx context.Context, key string) error {
	_, err := c.c.Evict(c.md5(key))
	return err
}

// Incr increases cached int-type value by given key as a counter.
func (c *SQLiteCacher) Incr(ctx context.Context, key string) error {
	var i int64
	err := c.read(key, &i)
	if err != nil {
		return err
	}

	if n, err := cache.Incr(i); err != nil {
		return err
	} else {
		return c.Put(ctx, key, n, 0)
	}
}

// Decr cached int value.
func (c *SQLiteCacher) Decr(ctx context.Context, key string) error {
	var i int64
	err := c.read(key, i)
	if err != nil {
		return err
	}

	if n, err := cache.Decr(i); err != nil {
		return err
	} else {
		return c.Put(ctx, key, n, 0)
	}
}

// IsExist returns true if cached value exists.
func (c *SQLiteCacher) IsExist(ctx context.Context, key string) (bool, error) {
	_, err := c.c.Get(c.md5(key))
	return cove.Hit(err)
}

// Flush deletes all cached data.
func (c *SQLiteCacher) Flush(ctx context.Context) error {
	_, err := c.c.EvictAll()
	return err
}

// StartAndGC starts GC routine based on config string settings.
func (c *SQLiteCacher) StartAndGC(ctx context.Context, opt cache.Options) (err error) {
	c.interval = opt.Interval
	ops := []cove.Op{
		//cove.DBRemoveOnClose(),
		//cove.WithTTL(time.Minute*10),
	}
	if c.interval > 0 {
		ops = append(ops, cove.WithVacuum(cove.Vacuum(time.Duration(c.interval)*time.Second, 1_000)))
	} else {
		ops = append(ops, cove.WithVacuum(nil))
	}
	if len(opt.AdapterConfig) == 0 {
		opt.AdapterConfig = filepath.Join(os.TempDir(), `admpub/cache.db`)
	}
	c.c, err = cove.New(cove.URIFromPath(opt.AdapterConfig), ops...)
	return err
}

func (c *SQLiteCacher) Close() error {
	c.interval = 0
	if c.c == nil {
		return nil
	}
	return c.c.Close()
}

func (c *SQLiteCacher) Client() interface{} {
	return c.c
}

func (c *SQLiteCacher) Name() string {
	return cacheEngineSQLite
}

const cacheEngineSQLite = `sqlite`

func AsClient(client interface{}) *cove.Cache {
	return client.(*cove.Cache)
}

func init() {
	cache.Register(cacheEngineSQLite, New())
}
