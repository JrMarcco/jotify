package sendstrategy

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
	"go.uber.org/zap"
)

var _ SendStrategy = (*DefaultSendStrategy)(nil)

type DefaultSendStrategy struct {
	logger *zap.Logger
}

func (s *DefaultSendStrategy) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	panic("not implemented")
}

func (s *DefaultSendStrategy) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	panic("not implemented")
}
