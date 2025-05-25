package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository/cache"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var _ cache.BizConfCache = (*BizConfRedisCache)(nil)

type BizConfRedisCache struct {
	client redis.Cmdable
	logger *zap.Logger
}

func (r *BizConfRedisCache) Set(ctx context.Context, id uint64, conf domain.BizConf) error {
	data, err := json.Marshal(conf)
	if err != nil {
		return fmt.Errorf("[jotify] marshal biz conf error: %w", err)
	}
	if err = r.client.Set(ctx, cache.BizConfCacheKey(id), data, cache.DefaultExpires).Err(); err != nil {
		return fmt.Errorf("[jotify] set biz conf to redis error: %w", err)
	}
	return nil
}

func (r *BizConfRedisCache) Get(ctx context.Context, id uint64) (domain.BizConf, error) {
	key := cache.BizConfCacheKey(id)
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// redis key 不存在
			return domain.BizConf{}, cache.ErrBizConfCacheKeyNotFound
		}
		return domain.BizConf{}, fmt.Errorf("[jotify] get biz conf from redis error: %w", err)
	}

	var bizConf domain.BizConf
	if err = json.Unmarshal([]byte(val), &bizConf); err != nil {
		return domain.BizConf{}, fmt.Errorf("[jotify] unmarshal biz conf error: %w", err)
	}
	return bizConf, nil
}

func NewBizConfRedisCache(rc redis.Cmdable, logger *zap.Logger) *BizConfRedisCache {
	return &BizConfRedisCache{
		client: rc,
		logger: logger,
	}
}
