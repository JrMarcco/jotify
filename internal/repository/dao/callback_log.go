package dao

import (
	"context"
	"time"

	"github.com/JrMarcco/jotify/internal/domain"
	"gorm.io/gorm"
)

type CallbackLog struct {
	NotificationId uint64
	RetriedTimes   int32
	NextRetryAt    int64
	Status         string
	CreatedAt      int64
	UpdatedAt      int64
}

type CallbackLogDAO interface {
	Find(ctx context.Context, startTime int64, startId uint64, batchSize int) ([]CallbackLog, uint64, error)
	Update(ctx context.Context, logs []CallbackLog) error
	FindByNotificationIds(ctx context.Context, notificationIds []uint64) ([]CallbackLog, error)
}

var _ CallbackLogDAO = (*DefaultCallbackLogDAO)(nil)

type DefaultCallbackLogDAO struct {
	db *gorm.DB
}

func (d *DefaultCallbackLogDAO) Find(ctx context.Context, startTime int64, startId uint64, batchSize int) ([]CallbackLog, uint64, error) {
	var logs []CallbackLog
	var nextStartId uint64

	res := d.db.WithContext(ctx).Model(&CallbackLog{}).
		Where("next_retry_at <= ?", startTime).
		Where("status = ?", domain.CallbackStatusPending).
		Where("id > ?", startId).
		Order("id ASC").
		Limit(batchSize).
		Find(&logs)

	if res.Error != nil {
		return nil, nextStartId, res.Error
	}

	if len(logs) > 0 {
		nextStartId = logs[len(logs)-1].NotificationId
	}

	return logs, nextStartId, nil
}

func (d *DefaultCallbackLogDAO) Update(ctx context.Context, logs []CallbackLog) error {
	if len(logs) == 0 {
		return nil
	}

	updateAt := time.Now().UnixMilli()
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, log := range logs {
			res := tx.Model(&CallbackLog{NotificationId: log.NotificationId}).
				Updates(map[string]any{
					"retried_times": log.RetriedTimes,
					"next_retry_at": log.NextRetryAt,
					"status":        log.Status,
					"updated_at":    updateAt,
				})

			if res.Error != nil {
				return res.Error
			}
		}
		return nil
	})
}

func (d *DefaultCallbackLogDAO) FindByNotificationIds(ctx context.Context, notificationIds []uint64) ([]CallbackLog, error) {
	var logs []CallbackLog
	err := d.db.WithContext(ctx).Model(&CallbackLog{}).Where("notification_id IN (?)", notificationIds).Find(&logs).Error
	return logs, err
}

func NewDefaultCallbackLogDAO(db *gorm.DB) *DefaultCallbackLogDAO {
	return &DefaultCallbackLogDAO{
		db: db,
	}
}
