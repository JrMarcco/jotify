package idempotent

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var _ Strategy = (*RedisStrategy)(nil)

// RedisStrategy 幂等策略的 redis 实现
type RedisStrategy struct {
	client  redis.Cmdable
	expires time.Duration
}

func (r *RedisStrategy) Exists(ctx context.Context, key string) (bool, error) {
	res, err := r.client.SetNX(ctx, r.redisKey(key), 1, r.expires).Result()
	if err != nil {
		return false, err
	}
	return !res, nil
}

func (r *RedisStrategy) MultiExists(ctx context.Context, keys []string) (map[string]bool, error) {
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

func (r *RedisStrategy) redisKey(bizKey string) string {
	return "idempotent:" + bizKey
}

func NewRedisStrategy(client redis.Cmdable, expires time.Duration) *RedisStrategy {
	return &RedisStrategy{
		client:  client,
		expires: expires,
	}
}
