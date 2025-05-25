package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository/cache/redis"
	"github.com/JrMarcco/jotify/internal/repository/dao"
	"go.uber.org/zap"
)

type NotificationRepo interface {
	Create(ctx context.Context, n domain.Notification) (domain.Notification, error)
	CreateWithCallback(ctx context.Context, n domain.Notification) (domain.Notification, error)
}

const (
	defaultQuota int32 = 1
)

var _ NotificationRepo = (*DefaultNotifRepo)(nil)

type DefaultNotifRepo struct {
	notifDAO   dao.NotificationDAO
	quotaCache *redis.QuotaCache
	logger     *zap.Logger
}

func (d *DefaultNotifRepo) Create(ctx context.Context, n domain.Notification) (domain.Notification, error) {
	// TODO 使用配额锁定记录表来进行配额管理
	// 扣减配额
	if err := d.quotaCache.Decr(ctx, n.BizId, n.Channel, defaultQuota); err != nil {
		return domain.Notification{}, err
	}

	entity, err := d.notifDAO.Create(ctx, d.toEntity(n))
	if err != nil {
		// 创建消息失败则退还额度
		if err := d.quotaCache.Incr(ctx, n.BizId, n.Channel, defaultQuota); err != nil {
			d.logger.Error(
				"[jotify] failed to refund quota",
				zap.Error(err),
				zap.Uint64("biz_id", n.BizId),
				zap.String("channel", string(n.Channel)),
			)
		}
		return domain.Notification{}, err
	}
	return d.toDomain(entity), nil
}

func (d *DefaultNotifRepo) CreateWithCallback(ctx context.Context, n domain.Notification) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
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

func NewDefaultNotifRepo(notifDAO dao.NotificationDAO, quotaCache *redis.QuotaCache, logger *zap.Logger) *DefaultNotifRepo {
	return &DefaultNotifRepo{
		notifDAO:   notifDAO,
		quotaCache: quotaCache,
		logger:     logger,
	}
}
