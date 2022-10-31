package redis

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app/server/registry"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var redisCli *redis.Client
var ctx = context.Background()

func init() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	redisCli = rdb
}

func TestRegister(t *testing.T) {
	tests := []struct {
		info    []*registry.Info
		wantErr bool
	}{
		{
			// set single info
			info: []*registry.Info{
				{
					ServiceName: "hertz.test.demo1",
					Addr:        utils.NewNetAddr("tcp", "127.0.0.1:8000"),
					Weight:      10,
					Tags:        nil,
				},
			},
			wantErr: false,
		},
		{
			// set multi infos
			info: []*registry.Info{
				{
					ServiceName: "hertz.test.demo2",
					Addr:        utils.NewNetAddr("tcp", "127.0.0.1:9000"),
					Weight:      15,
					Tags:        nil,
				},
				{
					ServiceName: "hertz.test.demo2",
					Addr:        utils.NewNetAddr("tcp", "127.0.0.1:9001"),
					Weight:      20,
					Tags:        nil,
				},
			},
			wantErr: false,
		},
	}
	for _, test := range tests {
		r := NewRedisRegistry("127.0.0.1:6379")
		for _, info := range test.info {
			if err := r.Register(info); err != nil {
				t.Errorf("info register err")
			}
			hash, err := prepareRegistryHash(info)
			assert.False(t, err != nil)
			redisCli.HGet(ctx, hash.key, hash.field)
		}
	}
}

// TestNewMentor test singleton
func TestNewMentor(t *testing.T) {
	m1 := newMentor()
	m2 := newMentor()
	assert.Equal(t, m1, m2)
}

func TestKeepAlive(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	rdb.Set(ctx, "k1", "v1", time.Second*20)
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Expire 是重设时间，不是延长时间
			rdb.Expire(ctx, "k1", time.Second*20)
		case <-ctx.Done():
			cancel()
		}
	}
}

func TestTTL(t *testing.T) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	rdb.Set(ctx, "k1", "v1", time.Second*10)
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			println(rdb.TTL(ctx, "k1").Val())
		}
	}
}

func TestForm(t *testing.T) {
	m := newMentor()
	m.insertForm("hertz", "127.0.0.1:8888")
	m.insertForm("hertz", "127.0.0.1:8080")
	m.insertForm("kitex", "127.0.0.1:9999")
	m.insertForm("volo", "127.0.0.1:7777")
	m.insertForm("volo", "127.0.0.1:6666")
	fmt.Println(m.mform)
	m.removeAddr("hertz", "127.0.0.1:8888")
	m.removeAddr("kitex", "127.0.0.1:9999")
	m.removeService("volo")
	fmt.Println(m.mform)
}
