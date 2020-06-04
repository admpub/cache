package main

import (
	"github.com/admpub/cache"
	_ "github.com/admpub/cache/redis"
)

func main() {
	ca, err := cache.Cacher(cache.Options{
		Adapter:       "redis",
		AdapterConfig: "addr=127.0.0.1:6379",
		OccupyMode:    true,
	})

	if err != nil {
		panic(err)
	}

	ca.Put("liyan", "cache", 60)
}
