package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

func main() {
	ctx := context.Background()
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{"192.168.157.133:6379", "192.168.157.133:6380", "192.168.157.133:6381"},

		// To route commands by latency or randomly, enable one of the following.
		//RouteByLatency: true,
		//RouteRandomly: true,
	})

	rdb.Set(ctx, "k1", "v1", 30*time.Second)
	fmt.Println(rdb.Get(ctx, "k1").Val())
}
