package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/JrMarcco/easy-kit/slice"
	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository/dao"
)

type CallbackLogRepo interface {
	Find(ctx context.Context, startTime int64, startId uint64, batchSize int) ([]domain.CallbackLog, uint64, error)
	Update(ctx context.Context, logs []domain.CallbackLog) error
	FindByNotificationIds(ctx context.Context, notificationIds []uint64) ([]domain.CallbackLog, error)
}

var _ CallbackLogRepo = (*DefaultCallbackLogRepo)(nil)

type DefaultCallbackLogRepo struct {
	notifDAO       dao.NotificationDAO
	callbackLogDAO dao.CallbackLogDAO
}

func (d *DefaultCallbackLogRepo) Find(ctx context.Context, startTime int64, startId uint64, batchSize int) ([]domain.CallbackLog, uint64, error) {
	entities, nextStartId, err := d.callbackLogDAO.Find(ctx, startTime, startId, batchSize)
	if err != nil {
		return nil, 0, err
	}

	if len(entities) < batchSize {
		nextStartId = 0
	}

	return slice.Map(entities, func(idx int, src dao.CallbackLog) domain.CallbackLog {
		n, _ := d.notifDAO.GetById(ctx, src.NotificationId)
		return d.toDomain(src, n)
	}), nextStartId, nil
}

func (d *DefaultCallbackLogRepo) Update(ctx context.Context, logs []domain.CallbackLog) error {
	return d.callbackLogDAO.Update(ctx, slice.Map(logs, func(_ int, src domain.CallbackLog) dao.CallbackLog {
		return d.toEntity(src)
	}))
}

func (d *DefaultCallbackLogRepo) FindByNotificationIds(ctx context.Context, notificationIds []uint64) ([]domain.CallbackLog, error) {
	logs, err := d.callbackLogDAO.FindByNotificationIds(ctx, notificationIds)
	if err != nil {
		return nil, err
	}

	ns, err := d.notifDAO.GetMapByIds(ctx, notificationIds)
	if err != nil {
		return nil, err
	}

	return slice.Map(logs, func(_ int, src dao.CallbackLog) domain.CallbackLog {
		return d.toDomain(src, ns[src.NotificationId])
	}), nil
}

func (d *DefaultCallbackLogRepo) toDomain(cl dao.CallbackLog, notification dao.Notification) domain.CallbackLog {
	return domain.CallbackLog{
		Notification: d.toDomainNotif(notification),
		RetriedTimes: cl.RetriedTimes,
		NextRetryAt:  cl.NextRetryAt,
		Status:       domain.CallbackLogStatus(cl.Status),
	}
}

func (d *DefaultCallbackLogRepo) toDomainNotif(entity dao.Notification) domain.Notification {
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

func (d *DefaultCallbackLogRepo) toEntity(cl domain.CallbackLog) dao.CallbackLog {
	return dao.CallbackLog{
		NotificationId: cl.Notification.Id,
		RetriedTimes:   cl.RetriedTimes,
		NextRetryAt:    cl.NextRetryAt,
		Status:         cl.Status.String(),
	}
}

func NewDefaultCallbackLogRepo(notifDAO dao.NotificationDAO, callbackLogDAO dao.CallbackLogDAO) *DefaultCallbackLogRepo {
	return &DefaultCallbackLogRepo{
		notifDAO:       notifDAO,
		callbackLogDAO: callbackLogDAO,
	}
}
