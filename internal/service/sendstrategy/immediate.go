package sendstrategy

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository"
)

var _ SendStrategy = (*ImmediateSendStrategy)(nil)

type ImmediateSendStrategy struct {
	notifRepo repository.NotificationRepo
}

func (s *ImmediateSendStrategy) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	panic("not implemented")
}

func (s *ImmediateSendStrategy) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	panic("not implemented")
}

func NewImmediateSendStrategy() *ImmediateSendStrategy {
	return &ImmediateSendStrategy{}
}
