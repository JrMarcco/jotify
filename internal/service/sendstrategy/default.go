package sendstrategy

import (
	"context"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/repository"
	"go.uber.org/zap"
)

var _ SendStrategy = (*DefaultSendStrategy)(nil)

// DefaultSendStrategy 默认消息发送策略
type DefaultSendStrategy struct {
	notifRepo   repository.NotificationRepo
	bizConfRepo repository.BizConfRepo
	logger      *zap.Logger
}

// Send 单条消息发送
func (dss *DefaultSendStrategy) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	n.SetSendTime()

	created, err := dss.create(ctx, n)
	if err != nil {
		return domain.SendResp{}, fmt.Errorf("[jotify] create delayed notification error: %w", err)
	}

	return domain.SendResp{
		Result: domain.SendResult{
			NotificationId: created.Id,
			Status:         created.Status,
		},
	}, nil
}

func (dss *DefaultSendStrategy) create(ctx context.Context, n domain.Notification) (domain.Notification, error) {
	if dss.needCallbackLog(ctx, n) {
		return dss.notifRepo.CreateWithCallback(ctx, n)
	}
	return dss.notifRepo.Create(ctx, n)
}

// BatchSend 批量发送
// 这里默认 ns 使用的是同一种发送策略
func (dss *DefaultSendStrategy) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	if len(ns) == 0 {
		return domain.BatchSendResp{}, fmt.Errorf("%w: notifications should not be empty", errs.ErrInvalidParam)
	}

	createdNs, err := dss.batchCreate(ctx, ns)
	if err != nil {
		return domain.BatchSendResp{}, fmt.Errorf("[jotify] create delayed notifications error: %w", err)
	}

	results := make([]domain.SendResult, 0, len(createdNs))
	for _, n := range createdNs {
		results = append(results, domain.SendResult{
			NotificationId: n.Id,
			Status:         n.Status,
		})
	}
	return domain.BatchSendResp{
		Results: results,
	}, nil
}

func (dss *DefaultSendStrategy) batchCreate(ctx context.Context, ns []domain.Notification) ([]domain.Notification, error) {
	const first = 0
	if dss.needCallbackLog(ctx, ns[first]) {
		return dss.notifRepo.BatchCreateWithCallback(ctx, ns)
	}
	return dss.notifRepo.BatchCreate(ctx, ns)
}

func (dss *DefaultSendStrategy) needCallbackLog(ctx context.Context, n domain.Notification) bool {
	bizConf, err := dss.bizConfRepo.GetById(ctx, n.BizId)
	if err != nil {
		dss.logger.Error("[jotify]get biz conf error", zap.Error(err))
		return false
	}
	return bizConf.CallbackConf != nil
}

func NewDefaultSendStrategy(
	notifRepo repository.NotificationRepo,
	bizConfRepo repository.BizConfRepo,
	logger *zap.Logger,
) *DefaultSendStrategy {
	return &DefaultSendStrategy{
		notifRepo:   notifRepo,
		bizConfRepo: bizConfRepo,
		logger:      logger,
	}
}
