package notification

import (
	"context"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/pkg/snowflake"
	"github.com/JrMarcco/jotify/internal/service/sendstrategy"
	"golang.org/x/sync/errgroup"
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
	if err := n.Validate(); err != nil {
		return domain.SendResp{}, err
	}

	n.Id = d.idGenerator.NextId(n.BizId, n.BizKey)
	// 替换消息发送策略为 Deadline，设置 1 分钟内发出消息
	n.ReplaceAsyncImmediate()
	return d.sendStrategy.Send(ctx, n)
}

func (d *DefaultSendService) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	resp := domain.BatchSendResp{}

	if len(ns) == 0 {
		return resp, fmt.Errorf("%w: no notifications to send", errs.ErrInvalidParam)
	}

	for _, n := range ns {
		if err := n.Validate(); err != nil {
			return resp, err
		}
		n.Id = d.idGenerator.NextId(n.BizId, n.BizKey)
	}

	sendResp, err := d.sendStrategy.BatchSend(ctx, ns)
	if err != nil {
		return resp, fmt.Errorf("%w: cause of: %w", errs.ErrFailedSendNotification, err)
	}

	resp.Results = sendResp.Results
	return resp, nil
}

func (d *DefaultSendService) BatchAsyncSend(ctx context.Context, ns []domain.Notification) (domain.BatchAsyncSendResp, error) {
	if len(ns) == 0 {
		return domain.BatchAsyncSendResp{}, fmt.Errorf("%w: no notifications to send", errs.ErrInvalidParam)
	}

	ids := make([]uint64, 0, len(ns))
	for _, n := range ns {
		if err := n.Validate(); err != nil {
			return domain.BatchAsyncSendResp{}, err
		}
		n.Id = d.idGenerator.NextId(n.BizId, n.BizKey)
		ids = append(ids, n.Id)

		n.ReplaceAsyncImmediate()
	}

	// 按照发送策略分组发送
	strategyGroup := make(map[string][]domain.Notification)
	for _, n := range ns {
		strategy := n.StrategyConfig.Type.String()
		strategyGroup[strategy] = append(strategyGroup[strategy], n)
	}

	// 分组发送
	eg, ctx := errgroup.WithContext(ctx)
	for _, group := range strategyGroup {
		notifications := group
		eg.Go(func() error {
			if _, err := d.sendStrategy.BatchSend(ctx, notifications); err != nil {
				return fmt.Errorf("%w: cause of: %w", errs.ErrFailedSendNotification, err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return domain.BatchAsyncSendResp{}, err
	}

	return domain.BatchAsyncSendResp{NotificationIds: ids}, nil
}
