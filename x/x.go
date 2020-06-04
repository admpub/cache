package x

import (
	"reflect"
	"sync"

	"github.com/admpub/cache"
)

// Cachex 缓存处理类
type Cachex struct {
	storage   cache.Cache
	querier   Querier
	sentinels sync.Map

	// useStale UseStaleWhenError
	useStale   bool
	defaultTTL int64
}

var DefaultTTL int64 = 86400 * 10 * 366

// New 新建缓存处理对象
func New(storage cache.Cache, querier Querier, defaultTTL ...int64) (c *Cachex) {
	c = &Cachex{
		storage:    storage,
		querier: 	querier,
		defaultTTL: DefaultTTL,
	}
	if len(defaultTTL) > 0 {
		c.defaultTTL = defaultTTL[0]
	}
	return c
}

// Get 获取
func (c *Cachex) Get(key string, value interface{}, opts ...GetOption) error {
	if v := reflect.ValueOf(value); v.Kind() != reflect.Ptr || v.IsNil() {
		panic("value not is non-nil pointer")
	}

	// 可选参数
	var options getOptions
	for _, opt := range opts {
		opt.apply(&options)
	}
	return c.get(key, value, options)
}

const (
	Disabled = -1
	Fresh = -2
)

// get ttl:-1 不用缓存; ttl:-2 强制更新缓存
func (c *Cachex) get(key string, value interface{}, options getOptions) error {
	querier := c.querier
	if options.querier != nil {
		querier = options.querier
	}
	if options.ttl == Disabled || options.disableCacheUsage {
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
	if ttl == Fresh || options.useFreshData {
		if querier == nil {
			return cache.ErrNotFound
		}
		err = querier.Query()
		if err != nil {
			return err
		}
		return c.storage.Put(key, value, ttl)
	}
	err = c.storage.Get(key, value)
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
	newSentinel := NewSentinel()
	actual, loaded := c.sentinels.LoadOrStore(key, newSentinel)
	sentinel := actual.(*Sentinel)
	if loaded {
		newSentinel.Close()
	} else {
		// 确保生产者总是能发出通知，并解锁
		defer c.sentinels.Delete(key)
		defer sentinel.CloseIfUnclose()
	}

	// 双重检查
	var staled interface{}
	err = c.storage.Get(key, value)
	switch err {
	case nil:
		if !loaded { // 将结果通知等待的过程
			sentinel.Done(reflect.ValueOf(value).Elem().Interface(), nil)
		}
		return nil
	case cache.ErrNotFound: // 下面查询
	case cache.ErrExpired: // 保存过期数据，如果下面查询失败，且useStale，返回过期数据
		staled = reflect.ValueOf(value).Elem().Interface()
	default:
		if !loaded { // 将错误通知等待的过程
			sentinel.Done(nil, err)
		}
		return err
	}

	if loaded {
		return sentinel.Wait(value)
	}
	err = querier.Query()
	if err != nil && c.useStale && staled != nil {
		// 当查询发生错误时，使用过期的缓存数据。该特性需要Storage支持
		reflect.ValueOf(value).Elem().Set(reflect.ValueOf(staled))
		sentinel.Done(staled, err)
		return err
	}

	if err != nil {
		sentinel.Done(nil, err)
		return err
	}

	// 更新到存储后端
	//elem := reflect.ValueOf(value).Elem().Interface()
	err = c.storage.Put(key, value, ttl)

	sentinel.Done(value, nil)
	return err
}

// Set 更新
func (c *Cachex) Set(key string, value interface{}, expire int64) error {
	return c.storage.Put(key, value, expire)
}

// Del 删除
func (c *Cachex) Del(keys ...string) error {
	for _, key := range keys {
		if err := c.storage.Delete(key); err != nil {
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
