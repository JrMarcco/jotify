package repository

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository/cache"
	"github.com/JrMarcco/jotify/internal/repository/dao"
	"go.uber.org/zap"
)

type BizConfRepo interface {
	GetById(ctx context.Context, id uint64) (domain.BizConf, error)
}

var _ BizConfRepo = (*DefaultBizConfRepo)(nil)

type DefaultBizConfRepo struct {
	dao        dao.BizConfDAO
	localCache cache.BizConfCache
	redisCache cache.BizConfCache
	logger     *zap.Logger
}

func (d *DefaultBizConfRepo) GetById(ctx context.Context, id uint64) (domain.BizConf, error) {
	// 从本地缓存获取
	bizConf, err := d.localCache.Get(ctx, id)
	if err == nil {
		return bizConf, nil
	}

	// 从 redis 获取
	bizConf, err = d.redisCache.Get(ctx, id)
	if err == nil {
		// 刷新本地缓存
		if lcErr := d.localCache.Set(ctx, id, bizConf); lcErr != nil {
			d.logger.Error("[jotify] failed to refresh biz conf local cache", zap.Error(lcErr), zap.Uint64("biz_id", id))
		}
		return bizConf, nil
	}

	bcEntity, err := d.dao.GetById(ctx, id)
	if err != nil {
		return domain.BizConf{}, err
	}

	bizConf = d.toDomain(bcEntity)

	// 先刷新本地缓存（本地缓存几乎不会出错）
	if lcErr := d.localCache.Set(ctx, id, bizConf); lcErr != nil {
		d.logger.Error("[jotify] failed to refresh biz conf local cache", zap.Error(lcErr), zap.Uint64("biz_id", id))
	}
	// 刷新 redis 缓存
	if rcErr := d.redisCache.Set(ctx, id, bizConf); rcErr != nil {
		d.logger.Error("[jotify] failed to refresh biz conf redis cache", zap.Error(rcErr), zap.Uint64("biz_id", id))
	}
	return bizConf, nil

}

func (d *DefaultBizConfRepo) toDomain(entity dao.BizConf) domain.BizConf {
	bizConf := domain.BizConf{
		Id:        entity.Id,
		OwnerId:   entity.OwnerId,
		OwnerType: entity.OwnerType,
		RateLimit: entity.RateLimit,
		CreateAt:  entity.CreatedAt,
		UpdateAt:  entity.UpdatedAt,
	}

	if entity.ChannelConf.Valid {
		bizConf.ChannelConf = &entity.ChannelConf.Val
	}

	if entity.TxNotifConf.Valid {
		bizConf.TxNotifConf = &entity.TxNotifConf.Val
	}

	if entity.QuotaConf.Valid {
		bizConf.QuotaConf = &entity.QuotaConf.Val
	}

	if entity.CallbackConf.Valid {
		bizConf.CallbackConf = &entity.CallbackConf.Val
	}

	return bizConf
}

func NewDefaultBizConfRepo(
	dao dao.BizConfDAO,
	localCache cache.BizConfCache,
	redisCache cache.BizConfCache,
	logger *zap.Logger,
) *DefaultBizConfRepo {
	return &DefaultBizConfRepo{
		dao:        dao,
		localCache: localCache,
		redisCache: redisCache,
		logger:     logger,
	}
}
