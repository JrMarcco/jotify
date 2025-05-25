package local

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository/cache"
	gcache "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var _ cache.BizConfCache = (*BizConfLocalCache)(nil)

type BizConfLocalCache struct {
	rc     redis.Cmdable
	gc     *gcache.Cache
	logger *zap.Logger
}

func (l *BizConfLocalCache) Set(_ context.Context, id uint64, conf domain.BizConf) error {
	l.gc.Set(cache.BizConfCacheKey(id), conf, cache.DefaultExpires)
	return nil
}

func (l *BizConfLocalCache) Get(_ context.Context, id uint64) (domain.BizConf, error) {
	key := cache.BizConfCacheKey(id)
	val, ok := l.gc.Get(key)
	if !ok {
		return domain.BizConf{}, cache.ErrBizConfCacheKeyNotFound
	}
	bizConf, ok := val.(domain.BizConf)
	if !ok {
		return domain.BizConf{}, cache.ErrWrongTypeOfBizConfCache
	}
	return bizConf, nil
}

func NewBizConfLocalCache(rc redis.Cmdable, gc *gcache.Cache, logger *zap.Logger) *BizConfLocalCache {
	lc := &BizConfLocalCache{
		rc:     rc,
		gc:     gc,
		logger: logger,
	}

	return lc
}
