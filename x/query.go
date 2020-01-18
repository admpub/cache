package x

// QueryFunc 查询过程签名
type QueryFunc func(key string, value interface{}) error

// Query 查询过程实现Querier接口
func (q QueryFunc) Query(key string, value interface{}) error {
	return q(key, value)
}

// Querier 查询接口
type Querier interface {
	// Query 查询. value必须是非nil指针,没找到返回NotFound错误实现
	Query(key string, value interface{}) error
}
