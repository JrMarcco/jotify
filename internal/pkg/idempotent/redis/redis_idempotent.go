package redis

import (
	"context"
	"time"

	"github.com/JrMarcco/jotify/internal/pkg/idempotent"
	"github.com/redis/go-redis/v9"
)

var _ idempotent.Strategy = (*Strategy)(nil)

// Strategy 幂等策略的 redis 实现
type Strategy struct {
	client  redis.Cmdable
	expires time.Duration
}

func (r *Strategy) Exists(ctx context.Context, key string) (bool, error) {
	res, err := r.client.SetNX(ctx, r.redisKey(key), 1, r.expires).Result()
	if err != nil {
		return false, err
	}
	return !res, nil
}

func (r *Strategy) MultiExists(ctx context.Context, keys []string) (map[string]bool, error) {
	pipeline := r.client.Pipeline()
	commands := make([]*redis.BoolCmd, len(keys))

	for i, key := range keys {
		commands[i] = pipeline.SetNX(ctx, r.redisKey(key), 1, r.expires)
	}

	_, er := pipeline.Exec(ctx)
	if er != nil {
		return nil, er
	}

	res := make(map[string]bool)
	for i, cmd := range commands {
		cmdRes, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		res[keys[i]] = !cmdRes
	}
	return res, nil
}

func (r *Strategy) redisKey(bizKey string) string {
	return "idempotent:" + bizKey
}

func NewStrategy(client redis.Cmdable, expires time.Duration) *Strategy {
	return &Strategy{
		client:  client,
		expires: expires,
	}
}
