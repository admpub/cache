package x

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/admpub/cache"
	"github.com/webx-top/com"
)

type TestData struct {
	Index   int
	Name    string
	Age     int
	IDs     []int
	Options map[string]interface{}
}

func TestX(t *testing.T) {
	storage := cache.NewMemoryCacher()
	c := New(storage, QueryFunc(func() error {
		return nil
	}))
	defer c.Release()
	ctx := context.Background()
	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := &TestData{}
			err := c.Get(ctx, `test`, data, Query(QueryFunc(func() error {
				data.Index = i
				data.Name = `test_` + strconv.Itoa(i)
				data.Age = 20
				data.IDs = []int{1, 23, 24}
				data.Options = map[string]interface{}{"test": true}
				time.Sleep(500 * time.Millisecond)
				return nil
			})))
			if err != nil {
				t.Error(err)
			}
			fmt.Printf("====[%d]====[%p]=============> \n%s\n\n", i, data, com.Dump(data, false))
		}(i)
	}
	wg.Wait()
	//panic(``)
}
