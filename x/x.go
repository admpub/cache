package x

import (
	"context"
	"reflect"
	"sync"

	"github.com/admpub/cache"
	"github.com/admpub/copier"
)

var DefaultTTL int64 = 86400 * 10 * 366

func NewFromPool(storage cache.Cache, querier Querier, defaultTTL ...int64) (c *Cachex) {
	c = cxPool.Get().(*Cachex)
	c.inpool = true
	c.Init(storage, querier, defaultTTL...)
	return c
}

// New 新建缓存处理对象
func New(storage cache.Cache, querier Querier, defaultTTL ...int64) (c *Cachex) {
	c = &Cachex{}
	c.Init(storage, querier, defaultTTL...)
	return c
}

// Cachex 缓存处理类
type Cachex struct {
	storage cache.Cache
	querier Querier
	inpool  bool

	// useStale UseStaleWhenError
	useStale   bool
	defaultTTL int64
}

var cxPool = sync.Pool{
	New: func() interface{} {
		return &Cachex{}
	},
}

func (c *Cachex) Reset() *Cachex {
	c.storage = nil
	c.querier = nil
	c.useStale = false
	c.defaultTTL = 0
	return c
}

func (c *Cachex) Release() {
	c.Reset()
	if c.inpool {
		cxPool.Put(c)
	}
}

func (c *Cachex) Init(storage cache.Cache, querier Querier, defaultTTL ...int64) *Cachex {
	c.storage = storage
	c.querier = querier
	c.useStale = false
	c.defaultTTL = DefaultTTL
	if len(defaultTTL) > 0 {
		c.defaultTTL = defaultTTL[0]
	}
	return c
}

// Get 获取
func (c *Cachex) Get(ctx context.Context, key string, value interface{}, opts ...GetOption) error {
	if v := reflect.ValueOf(value); v.Kind() != reflect.Ptr || v.IsNil() {
		panic("value not is non-nil pointer")
	}

	// 可选参数
	var options Options
	for _, opt := range opts {
		opt(&options)
	}
	return c.get(ctx, key, value, options)
}

// get ttl:-1 不用缓存; ttl:-2 强制更新缓存
func (c *Cachex) get(ctx context.Context, key string, value interface{}, options Options) error {
	querier := c.querier
	if options.querier != nil {
		querier = options.querier
	}
	if options.disableCacheUsage {
		if querier == nil {
			return cache.ErrNotFound
		}
		return querier.Query()
	}
	var (
		ttl int64
		err error
	)
	if options.ttl != 0 {
		ttl = options.ttl
	}
	if ttl == 0 {
		ttl = c.defaultTTL
	}
	if options.useFreshData {
		if querier == nil {
			return cache.ErrNotFound
		}
		err = querier.Query()
		if err != nil {
			return err
		}
		return c.storage.Put(ctx, key, value, ttl)
	}
	err = c.storage.Get(ctx, key, value)
	switch err {
	case cache.ErrNotFound, cache.ErrExpired: // 下面查询
	default:
		return err
	}
	if querier == nil {
		return cache.ErrNotFound
	}
	// 在一份实例中
	// 不同时发起重复的查询请求——解决缓存失效风暴
	getValue, getErr, _ := c.storage.Do(key, func() (interface{}, error) {
		dErr := c.storage.Get(ctx, key, value)
		if dErr == nil {
			return value, dErr
		}
		var staled interface{}
		switch dErr {
		case cache.ErrNotFound: // 下面查询
		case cache.ErrExpired: // 保存过期数据，如果下面查询失败，且useStale，返回过期数据
			staled = reflect.ValueOf(value).Elem().Interface()
		default:
			return value, dErr
		}
		dErr = querier.Query()
		if dErr != nil {
			if c.useStale && staled != nil { // 当查询发生错误时，使用过期的缓存数据。该特性需要Storage支持
				reflect.ValueOf(value).Elem().Set(reflect.ValueOf(staled))
				return staled, dErr
			}
			return value, dErr
		}
		// 更新到存储后端
		dErr = c.storage.Put(ctx, key, value, ttl)
		return value, dErr
	})
	if getErr != nil {
		return err
	}
	if getValue == value {
		return nil
	}
	return copier.Copy(value, getValue)
}

// Set 更新
func (c *Cachex) Set(ctx context.Context, key string, value interface{}, expire int64) error {
	return c.storage.Put(ctx, key, value, expire)
}

// Del 删除
func (c *Cachex) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := c.storage.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// UseStaleWhenError 设置当查询发生错误时，使用过期的缓存数据。该特性需要Storage支持（Get返回过期的缓存数据和Expired错误实现）。默认关闭。
func (c *Cachex) UseStaleWhenError(use bool) *Cachex {
	c.useStale = use
	return c
}
