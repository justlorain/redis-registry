package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app/client/discovery"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/go-redis/redis/v8"
)

var _ discovery.Resolver = (*redisResolver)(nil)

type redisResolver struct {
	client *redis.Client
}

func NewRedisResolver(addr string, opts ...Option) discovery.Resolver {
	redisOpts := &redis.Options{Addr: addr}
	for _, opt := range opts {
		opt(redisOpts)
	}
	rdb := redis.NewClient(redisOpts)
	return &redisResolver{
		client: rdb,
	}
}

func (r *redisResolver) Target(_ context.Context, target *discovery.TargetInfo) string {
	return target.Host
}

// Resolve 返回实例数组
// 写入127.0.0.1:8888的逻辑，写入127.0.0.1:8888就可以成功调用，所以核心在于如何通过服务名获取服务地址
func (r *redisResolver) Resolve(ctx context.Context, desc string) (discovery.Result, error) {
	rdb := r.client
	fvs := rdb.HGetAll(ctx, fmt.Sprintf("/%s/%s/%s", Hertz, desc, Server)).Val()
	var (
		ri  registryInfo
		its []discovery.Instance
	)
	// f: addr; v: JSON
	for f, v := range fvs {
		err := json.Unmarshal([]byte(v), &ri)
		if err != nil {
			hlog.Warnf("HERTZ: fail to unmarshal with err: %v, ignore instance addr: %v", err, f)
		}
		weight := ri.weight
		if weight <= 0 {
			weight = DefaultWeight
		}
		its = append(its, discovery.NewInstance(TCP, ri.addr, weight, ri.tags))
	}
	return discovery.Result{
		CacheKey:  desc,
		Instances: its,
	}, nil
}

func (r *redisResolver) Name() string {
	return Redis
}
