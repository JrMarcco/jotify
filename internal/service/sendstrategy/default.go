package sendstrategy

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
)

var _ SendStrategy = (*DefaultSendStrategy)(nil)

type DefaultSendStrategy struct {
}

func (s *DefaultSendStrategy) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	panic("not implemented")
}

func (s *DefaultSendStrategy) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	panic("not implemented")
}
