package x

// QueryFunc 查询过程签名
type QueryFunc func(recv interface{}) error

// Query 查询过程实现Querier接口
func (q QueryFunc) Query(recv interface{}) error {
	return q(recv)
}

// Querier 查询接口
type Querier interface {
	// Query 查询. value必须是非nil指针,没找到返回NotFound错误实现
	Query(recv interface{}) error
}
