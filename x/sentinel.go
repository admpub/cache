package x

import (
	"reflect"

	"github.com/admpub/copier"
	"github.com/admpub/cache"
)

// Sentinel 哨兵。一个生产者，多个消费者等待生产者完成并提交结果
type Sentinel struct {
	flag chan interface{}

	result interface{}
	err    error
}

// NewSentinel 新建哨兵
func NewSentinel() *Sentinel {
	return &Sentinel{
		flag: make(chan interface{}),
	}
}

// Done 生产者提交结果，消费者将等待到提交的结果。result必须是具体数据变量的接口
func (s *Sentinel) Done(result interface{}, err error) error {
	value := reflect.ValueOf(result)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if result != nil {
		newResult := reflect.New(value.Type()).Interface()
		e := copier.Copy(newResult, result)
		if e != nil {
			return e
		}

		s.result = newResult
	}
	s.err = err

	close(s.flag)
	return nil
}

// Wait 消费者等待生产者提交结果。result必须是一个指针的接口
func (s *Sentinel) Wait(result interface{}) error {
	if value := reflect.ValueOf(result); value.Kind() != reflect.Ptr || value.IsNil() {
		panic("value must is a non-nil pointer")
	}

	<-s.flag

	if s.err != nil {
		return s.err
	}

	if s.result != nil {
		err := copier.Copy(result, s.result)
		if err != nil {
			return err
		}
	} else if s.err == nil {
		return cache.ErrNotFound
	}

	return nil
}

// Close 直接关闭。重复关闭内部channel会导致panic。
func (s *Sentinel) Close() {
	close(s.flag)
}

// CloseIfUnclose 如果未关闭则关闭。
func (s *Sentinel) CloseIfUnclose() {
	select {
	case <-s.flag:
	default:
		close(s.flag)
	}
}
