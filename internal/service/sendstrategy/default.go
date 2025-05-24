package sendstrategy

import (
	"context"

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
	panic("not implemented")
}

func (dss *DefaultSendStrategy) createNotifRecord(ctx context.Context, n domain.Notification) (domain.Notification, error) {
	panic("not implemented")
}

func (dss *DefaultSendStrategy) needCallbackLog(ctx context.Context, n domain.Notification) bool {
	bizConf, err := dss.bizConfRepo.GetById(ctx, n.BizId)
	if err != nil {
		dss.logger.Error("[jotify]get biz conf error", zap.Error(err))
		return false
	}
	return bizConf.CallbackConfig != nil
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
