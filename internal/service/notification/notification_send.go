package notification

import (
	"context"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/pkg/snowflake"
	"github.com/JrMarcco/jotify/internal/service/sendstrategy"
)

//go:generate mockGen -source=./notification_send.go -destination=./mock/send_service.mock.go -package=notificationmock -type=SendService
type SendService interface {
	Send(ctx context.Context, n domain.Notification) (domain.SendResp, error)
	AsyncSend(ctx context.Context, n domain.Notification) (domain.SendResp, error)
	BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error)
	BatchAsyncSend(ctx context.Context, ns []domain.Notification) (domain.BatchAsyncSendResp, error)
}

var _ SendService = (*DefaultSendService)(nil)

type DefaultSendService struct {
	idGenerator  *snowflake.Generator
	sendStrategy sendstrategy.SendStrategy
}

func (d *DefaultSendService) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	resp := domain.SendResp{
		Result: domain.SendResult{
			Status: domain.SendStatusFailed,
		},
	}

	if err := n.Validate(); err != nil {
		return resp, err
	}

	n.Id = d.idGenerator.NextId(n.BizId, n.BizKey)
	sendResp, err := d.sendStrategy.Send(ctx, n)
	if err != nil {
		return resp, fmt.Errorf("%w: cause of: %w", errs.ErrFailedSendNotification, err)
	}
	return sendResp, nil
}

func (d *DefaultSendService) AsyncSend(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DefaultSendService) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DefaultSendService) BatchAsyncSend(ctx context.Context, ns []domain.Notification) (domain.BatchAsyncSendResp, error) {
	//TODO implement me
	panic("implement me")
}
