package main

import (
	"context"

	"github.com/admpub/cache"
	_ "github.com/admpub/cache/redis"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ca, err := cache.Cacher(ctx, cache.Options{
		Adapter:       "redis",
		AdapterConfig: "addr=127.0.0.1:6379",
		OccupyMode:    true,
	})

	if err != nil {
		panic(err)
	}

	ca.Put(ctx, "key", "cache", 60)
}
