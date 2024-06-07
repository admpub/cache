package x

// Options Get方法的可选参数项
type Options struct {
	querier Querier
	ttl     int64 // seconds

	// DisableCacheUsage disables the cache.
	// It can be useful during debugging.
	disableCacheUsage bool

	// UseFreshData will ignore content in the cache and always pull fresh data.
	// The pulled data will subsequently be saved in the cache.
	useFreshData bool
}

func (o *Options) SetQuerier(querier Querier) {
	o.querier = querier
}

func (o *Options) SetTTL(ttl int64) {
	o.ttl = ttl
}

func (o *Options) AddTTL(ttl int64) {
	o.ttl += ttl
}

func (o *Options) SetDisableCacheUsage(disableCacheUsage bool) {
	o.disableCacheUsage = disableCacheUsage
}

func (o *Options) SetUseFreshData(useFreshData bool) {
	o.useFreshData = useFreshData
}

// GetOption Get方法的可选参数项结构，不需要直接调用。
type GetOption func(*Options)

// Query 为Get操作定制查询过程
func Query(querier Querier) GetOption {
	return func(o *Options) {
		o.SetQuerier(querier)
	}
}

// TTL 为Get操作定制TTL
func TTL(ttl int64) GetOption {
	return func(o *Options) {
		o.SetTTL(ttl)
	}
}

func AddTTL(ttl int64) GetOption {
	return func(o *Options) {
		o.AddTTL(ttl)
	}
}

func DisableCacheUsage(disableCacheUsage bool) GetOption {
	return func(o *Options) {
		o.SetDisableCacheUsage(disableCacheUsage)
	}
}

func UseFreshData(useFreshData bool) GetOption {
	return func(o *Options) {
		o.SetUseFreshData(useFreshData)
	}
}
