package redis

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app/server/registry"
	"github.com/go-redis/redis/v8"
	"sync"
)

var _ registry.Registry = (*redisRegistry)(nil)

type redisRegistry struct {
	client *redis.Client
	rctx   *registryContext
	mu     sync.Mutex
	wg     sync.WaitGroup
}

type registryContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRedisRegistry create a redis registry
func NewRedisRegistry(addr string, opts ...Option) registry.Registry {
	redisOpts := &redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	}
	for _, opt := range opts {
		opt(redisOpts)
	}
	rdb := redis.NewClient(redisOpts)
	return &redisRegistry{
		client: rdb,
	}
}

// Register 注册时调用
// 注意：redis 节点和 服务地址都有可能有多个
func (r *redisRegistry) Register(info *registry.Info) error {
	// 参数校验
	if err := validateRegistryInfo(info); err != nil {
		return err
	}
	rctx := registryContext{}
	rctx.ctx, rctx.cancel = context.WithCancel(context.Background())
	m := newMentor()
	r.wg.Add(1)
	go m.subscribe(rctx.ctx, info, r)
	r.wg.Wait()

	// 获取 client
	rdb := r.client
	hash, err := prepareRegistryHash(info)
	if err != nil {
		return err
	}
	// 设置 key 并发布消息
	r.mu.Lock()
	r.rctx = &rctx
	rdb.HSet(rctx.ctx, hash.key, hash.field, hash.value)
	rdb.Expire(rctx.ctx, hash.key, DefaultExpireTime)
	// 发布消息，通知 mentor 保存到 form 中
	rdb.Publish(rctx.ctx, hash.key, fmt.Sprintf("%s-%s-%s", Register, info.ServiceName, info.Addr.String()))
	r.mu.Unlock()

	// 定期延长
	go m.monitorTTL(rctx.ctx, hash, info, r)
	go keepAlive(rctx.ctx, hash, r)
	return nil
}

// Deregister 下线时调用
func (r *redisRegistry) Deregister(info *registry.Info) error {
	if err := validateRegistryInfo(info); err != nil {
		return err
	}
	rctx := r.rctx
	rdb := r.client
	hash, err := prepareRegistryHash(info)
	if err != nil {
		return err
	}
	r.mu.Lock()
	rdb.HDel(rctx.ctx, hash.key, hash.field)
	rdb.Publish(rctx.ctx, hash.key, fmt.Sprintf("%s-%s-%s", Deregister, info.ServiceName, info.Addr.String()))
	rctx.cancel()
	r.mu.Unlock()
	return nil
}
