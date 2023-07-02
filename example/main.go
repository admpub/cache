package main

import (
	"context"

	"github.com/admpub/cache"
	_ "github.com/admpub/cache/redis"
)

func main() {
	ctx := context.Background()
	ca, err := cache.Cacher(ctx, cache.Options{
		Adapter:       "redis",
		AdapterConfig: "addr=127.0.0.1:6379",
		OccupyMode:    true,
	})

	if err != nil {
		panic(err)
	}

	ca.Put(ctx, "liyan", "cache", 60)
}
