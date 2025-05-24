package local

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository/cache"
	gcache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

var _ cache.BizConfCache = (*BizConfLocalCache)(nil)

type BizConfLocalCache struct {
	c      gcache.Cache
	logger *zap.Logger
}

func (lc *BizConfLocalCache) Get(ctx context.Context, id uint64) (domain.BizConf, error) {
	//TODO implement me
	panic("implement me")
}

func (lc *BizConfLocalCache) Set(ctx context.Context, id uint64, conf domain.BizConf) error {
	//TODO implement me
	panic("implement me")
}
