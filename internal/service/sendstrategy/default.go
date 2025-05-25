package sendstrategy

import (
	"context"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository"
	"go.uber.org/zap"
)

var _ SendStrategy = (*DefaultSendStrategy)(nil)

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

func (dss *DefaultSendStrategy) needCallbackLog(ctx context.Context, n domain.Notification) bool {
	bizConf, err := dss.bizConfRepo.GetById(ctx, n.BizId)
	if err != nil {
		dss.logger.Error("[jotify]get biz conf error", zap.Error(err))
		return false
	}
	return bizConf.CallbackConf != nil
}

func (dss *DefaultSendStrategy) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	panic("not implemented")
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
