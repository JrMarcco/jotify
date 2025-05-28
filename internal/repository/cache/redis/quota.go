package redis

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/repository/cache"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	//go:embed lua/quota_incr.lua
	quotaIncrLua string
	//go:embed lua/quota_bach_incr.lua
	quotaBatchIncrLua string
	//go:embed lua/quota_bach_decr.lua
	quotaBatchDecrLua string
)

var _ cache.QuotaCache = (*QuotaRedisCache)(nil)

type QuotaRedisCache struct {
	client redis.Cmdable
	logger *zap.Logger
}

func (q *QuotaRedisCache) Incr(ctx context.Context, param cache.QuotaParam) error {
	keys := []string{
		q.redisKey(param.BizId, param.Channel),
	}
	return q.client.Eval(ctx, quotaIncrLua, keys, param.Quota).Err()
}

func (q *QuotaRedisCache) BatchIncr(ctx context.Context, params []cache.QuotaParam) error {
	if len(params) == 0 {
		return nil
	}

	keys, args := q.redisKeysAndArgs(params)
	return q.client.Eval(ctx, quotaBatchIncrLua, keys, args...).Err()
}

func (q *QuotaRedisCache) Decr(ctx context.Context, param cache.QuotaParam) error {
	res, err := q.client.DecrBy(ctx, q.redisKey(param.BizId, param.Channel), int64(param.Quota)).Result()
	if err != nil {
		return err
	}

	if res < 0 {
		q.logger.Error("insufficient quota", zap.Uint64("bizId", param.BizId), zap.String("channel", string(param.Channel)))
		return errs.ErrInsufficientQuota
	}
	return nil
}

func (q *QuotaRedisCache) BatchDecr(ctx context.Context, params []cache.QuotaParam) error {
	if len(params) == 0 {
		return nil
	}

	keys, args := q.redisKeysAndArgs(params)
	res, err := q.client.Eval(ctx, quotaBatchDecrLua, keys, args...).Result()
	if err != nil {
		return err
	}

	resMsg, ok := res.(string)
	if !ok {
		return fmt.Errorf("[jotify] wrong type of redis eval result")
	}

	if resMsg == "" {
		return nil
	}
	return fmt.Errorf("[jotify] the quota of %s is not enough", resMsg)
}

func (q *QuotaRedisCache) redisKey(bizId uint64, channel domain.Channel) string {
	return fmt.Sprintf("quota:%d:%s", bizId, channel)
}

func (q *QuotaRedisCache) redisKeysAndArgs(params []cache.QuotaParam) ([]string, []any) {
	keys := make([]string, 0, len(params))
	args := make([]any, 0, len(params))
	for _, param := range params {
		keys = append(keys, q.redisKey(param.BizId, param.Channel))
		args = append(args, param.Quota)
	}
	return keys, args
}

func NewQuotaRedisCache(rc redis.Cmdable, logger *zap.Logger) *QuotaRedisCache {
	return &QuotaRedisCache{
		client: rc,
		logger: logger,
	}
}
