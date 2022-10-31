package redis

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app/server/registry"
	"github.com/go-redis/redis/v8"
	"net"
	"time"
)

const (
	Register   = "register"
	Deregister = "deregister"
	Hertz      = "hertz"
	Mentor     = "mentor"
	Server     = "server"
	Client     = "client"
	Redis      = "redis"
	TCP        = "tcp"
)

const (
	DefaultExpireTime    = time.Second * 60
	DefaultTickerTime    = time.Second * 30
	DefaultKeepAliveTime = time.Second * 60
	DefaultMonitorTime   = time.Second * 30
	DefaultWeight        = 10
)

type Option func(opts *redis.Options)

func WithPassword(password string) Option {
	return func(opts *redis.Options) {
		opts.Password = password
	}
}

func WithDB(db int) Option {
	return func(opts *redis.Options) {
		opts.DB = db
	}
}

func WithTLSConfig(t *tls.Config) Option {
	return func(opts *redis.Options) {
		opts.TLSConfig = t
	}
}

func WithDialer(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(opts *redis.Options) {
		opts.Dialer = dialer
	}
}

func WithReadTimeout(t time.Duration) Option {
	return func(opts *redis.Options) {
		opts.ReadTimeout = t
	}
}

func WithWriteTimeout(t time.Duration) Option {
	return func(opts *redis.Options) {
		opts.WriteTimeout = t
	}
}

type registryHash struct {
	key   string
	field string
	value string
}

type registryInfo struct {
	ServiceName string
	Addr        string
	Weight      int
	Tags        map[string]string
}

// validateRegistryInfo Validate the registry.Info
func validateRegistryInfo(info *registry.Info) error {
	if info == nil {
		return fmt.Errorf("registry.Info can not be empty")
	}
	if info.ServiceName == "" {
		return fmt.Errorf("registry.Info ServiceName can not be empty")
	}
	if info.Addr == nil {
		return fmt.Errorf("registry.Info Addr can not be empty")
	}
	return nil
}

func prepareRegistryHash(info *registry.Info) (*registryHash, error) {
	meta, err := json.Marshal(convertInfo(info))
	if err != nil {
		return nil, err
	}
	return &registryHash{
		// /hertz/service-name/service-type
		key:   fmt.Sprintf("/hertz/%s/%s", info.ServiceName, Server),
		field: info.Addr.String(),
		value: string(meta),
	}, nil
}

func convertInfo(info *registry.Info) *registryInfo {
	return &registryInfo{
		ServiceName: info.ServiceName,
		Addr:        info.Addr.String(),
		Weight:      info.Weight,
		Tags:        info.Tags,
	}
}

func keepAlive(ctx context.Context, hash *registryHash, r *redisRegistry) {
	ticker := time.NewTicker(DefaultTickerTime)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.client.Expire(ctx, hash.key, DefaultKeepAliveTime)
		case <-ctx.Done():
			break
		default:
		}
	}
}
