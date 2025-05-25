package redis

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	//go:embed lua/quota_incr.lua
	quotaIncrLua string
)

type QuotaCache struct {
	client redis.Cmdable
	logger *zap.Logger
}

func (q *QuotaCache) Incr(ctx context.Context, bizId uint64, channel domain.Channel, quota int32) error {
	keys := []string{
		q.redisKey(bizId, channel),
	}
	return q.client.Eval(ctx, quotaIncrLua, keys, quota).Err()
}

func (q *QuotaCache) Decr(ctx context.Context, bizId uint64, channel domain.Channel, quota int32) error {
	res, err := q.client.DecrBy(ctx, q.redisKey(bizId, channel), int64(quota)).Result()
	if err != nil {
		return err
	}

	if res < 0 {
		q.logger.Error("insufficient quota", zap.Uint64("bizId", bizId), zap.String("channel", string(channel)))
		return errs.ErrInsufficientQuota
	}
	return nil
}

func (q *QuotaCache) redisKey(bizId uint64, channel domain.Channel) string {
	return fmt.Sprintf("quota:%d:%s", bizId, channel)
}

func NewQuotaCache(rc redis.Cmdable, logger *zap.Logger) *QuotaCache {
	return &QuotaCache{
		client: rc,
		logger: logger,
	}
}
