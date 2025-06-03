package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/JrMarcco/easy-kit/slice"
	"github.com/JrMarcco/easy-kit/xmap"
	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository/cache"
	"github.com/JrMarcco/jotify/internal/repository/cache/redis"
	"github.com/JrMarcco/jotify/internal/repository/dao"
	"go.uber.org/zap"
)

type NotificationRepo interface {
	Create(ctx context.Context, n domain.Notification) (domain.Notification, error)
	CreateWithCallback(ctx context.Context, n domain.Notification) (domain.Notification, error)
	BatchCreate(ctx context.Context, ns []domain.Notification) ([]domain.Notification, error)
	BatchCreateWithCallback(ctx context.Context, ns []domain.Notification) ([]domain.Notification, error)
	MarkSuccess(ctx context.Context, n domain.Notification) error
	MarkFailure(ctx context.Context, n domain.Notification) error
}

const (
	defaultQuota int32 = 1
)

var _ NotificationRepo = (*DefaultNotifRepo)(nil)

type DefaultNotifRepo struct {
	notifDAO   dao.NotificationDAO
	quotaCache *redis.QuotaRedisCache
	logger     *zap.Logger
}

func (d *DefaultNotifRepo) Create(ctx context.Context, n domain.Notification) (domain.Notification, error) {
	// TODO 使用配额锁定记录表来进行配额管理
	// 扣减配额
	quotaParam := cache.QuotaParam{
		BizId:   n.BizId,
		Channel: n.Channel,
		Quota:   defaultQuota,
	}
	if err := d.quotaCache.Decr(ctx, quotaParam); err != nil {
		return domain.Notification{}, err
	}

	entity, err := d.notifDAO.Create(ctx, d.toEntity(n))
	if err != nil {
		// 创建消息失败则退还配额
		if refundErr := d.quotaCache.Incr(ctx, quotaParam); refundErr != nil {
			d.logger.Error(
				"[jotify] failed to refund quota",
				zap.Error(refundErr),
				zap.Uint64("biz_id", n.BizId),
				zap.String("channel", string(n.Channel)),
			)
		}
		return domain.Notification{}, err
	}
	return d.toDomain(entity), nil
}

func (d *DefaultNotifRepo) CreateWithCallback(ctx context.Context, n domain.Notification) (domain.Notification, error) {
	// TODO 使用配额锁定记录表来进行配额管理
	// 扣减配额
	quotaParam := cache.QuotaParam{
		BizId:   n.BizId,
		Channel: n.Channel,
		Quota:   defaultQuota,
	}
	if err := d.quotaCache.Decr(ctx, quotaParam); err != nil {
		return domain.Notification{}, err
	}

	entity, err := d.notifDAO.CreateWithCallback(ctx, d.toEntity(n))
	if err != nil {
		// 创建消息失败则退还配额
		if refundErr := d.quotaCache.Incr(ctx, quotaParam); refundErr != nil {
			d.logger.Error(
				"[jotify] failed to refund quota",
				zap.Error(refundErr),
				zap.Uint64("biz_id", n.BizId),
				zap.String("channel", string(n.Channel)),
			)
		}
		return domain.Notification{}, err
	}
	return d.toDomain(entity), nil
}

func (d *DefaultNotifRepo) BatchCreate(ctx context.Context, ns []domain.Notification) ([]domain.Notification, error) {
	if len(ns) == 0 {
		return nil, nil
	}
	// 扣减库存
	// TODO 使用配额锁定记录表来进行配额管理
	quotaParams := d.buildQuotaParams(ns)
	if err := d.quotaCache.BatchDecr(ctx, quotaParams); err != nil {
		return nil, err
	}

	entities, err := d.notifDAO.BatchCreate(ctx, d.toEntities(ns))
	if err != nil {
		// 创建消息失败则退还配额
		if refundErr := d.quotaCache.BatchIncr(ctx, quotaParams); refundErr != nil {
			d.logger.Error("[jotify] failed to batch refund quota", zap.Error(refundErr))
		}
	}
	return slice.Map(entities, func(_ int, entity dao.Notification) domain.Notification {
		return d.toDomain(entity)
	}), nil
}

func (d *DefaultNotifRepo) BatchCreateWithCallback(ctx context.Context, ns []domain.Notification) ([]domain.Notification, error) {
	if len(ns) == 0 {
		return nil, nil
	}
	// 扣减库存
	// TODO 使用配额锁定记录表来进行配额管理
	quotaParams := d.buildQuotaParams(ns)
	if err := d.quotaCache.BatchDecr(ctx, quotaParams); err != nil {
		return nil, err
	}

	entities, err := d.notifDAO.BatchCreateWithCallback(ctx, d.toEntities(ns))
	if err != nil {
		// 创建消息失败则退还配额
		if refundErr := d.quotaCache.BatchIncr(ctx, quotaParams); refundErr != nil {
			d.logger.Error("[jotify] failed to batch refund quota", zap.Error(refundErr))
		}
	}
	return slice.Map(entities, func(_ int, entity dao.Notification) domain.Notification {
		return d.toDomain(entity)
	}), nil
}

func (d *DefaultNotifRepo) MarkSuccess(ctx context.Context, n domain.Notification) error {
	return d.notifDAO.MarkSuccess(ctx, d.toEntity(n))
}

func (d *DefaultNotifRepo) MarkFailure(ctx context.Context, n domain.Notification) error {
	err := d.notifDAO.MarkFailure(ctx, d.toEntity(n))
	if err != nil {
		return err
	}
	return d.quotaCache.Incr(ctx, cache.QuotaParam{
		BizId:   n.BizId,
		Channel: n.Channel,
		Quota:   defaultQuota,
	})
}

func (d *DefaultNotifRepo) buildQuotaParams(ns []domain.Notification) []cache.QuotaParam {
	m := make(map[string]cache.QuotaParam)
	for _, n := range ns {
		key := fmt.Sprintf("%s:%s", n.BizId, n.Channel.String())
		param, ok := m[key]
		if !ok {
			param = cache.QuotaParam{
				BizId:   n.BizId,
				Channel: n.Channel,
			}
		}
		param.Quota++
		m[key] = param
	}
	return xmap.Vals(m)
}

func (d *DefaultNotifRepo) toEntity(n domain.Notification) dao.Notification {
	tplParams, _ := n.MarshalTemplateParams()
	receivers, _ := n.MarshalReceivers()
	return dao.Notification{
		Id:            n.Id,
		BizId:         n.BizId,
		BizKey:        n.BizKey,
		Receivers:     receivers,
		Channel:       n.Channel.String(),
		TplId:         n.Template.Id,
		TplVersionId:  n.Template.VersionId,
		TplParams:     tplParams,
		Status:        n.Status.String(),
		ScheduleStrat: n.ScheduledStart.UnixMilli(),
		ScheduleEnd:   n.ScheduledEnd.UnixMilli(),
		Version:       n.Version,
	}
}

func (d *DefaultNotifRepo) toEntities(ns []domain.Notification) []dao.Notification {
	return slice.Map(ns, func(_ int, n domain.Notification) dao.Notification {
		return d.toEntity(n)
	})
}

func (d *DefaultNotifRepo) toDomain(entity dao.Notification) domain.Notification {
	var tplParams map[string]string
	_ = json.Unmarshal([]byte(entity.TplParams), &tplParams)

	var receivers []string
	_ = json.Unmarshal([]byte(entity.Receivers), &receivers)

	return domain.Notification{
		Id:        entity.Id,
		BizId:     entity.BizId,
		BizKey:    entity.BizKey,
		Receivers: receivers,
		Channel:   domain.Channel(entity.Channel),
		Template: domain.Template{
			Id:        entity.TplId,
			VersionId: entity.TplVersionId,
			Params:    tplParams,
		},
		Status:         domain.SendStatus(entity.Status),
		ScheduledStart: time.UnixMilli(entity.ScheduleStrat),
		ScheduledEnd:   time.UnixMilli(entity.ScheduleEnd),
		Version:        entity.Version,
	}
}

func NewDefaultNotifRepo(notifDAO dao.NotificationDAO, quotaCache *redis.QuotaRedisCache, logger *zap.Logger) *DefaultNotifRepo {
	return &DefaultNotifRepo{
		notifDAO:   notifDAO,
		quotaCache: quotaCache,
		logger:     logger,
	}
}
