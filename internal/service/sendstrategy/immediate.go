package sendstrategy

import (
	"context"
	"errors"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/service/sender"
)

var _ SendStrategy = (*ImmediateSendStrategy)(nil)

type ImmediateSendStrategy struct {
	notifRepo repository.NotificationRepo
	sender    sender.Sender
}

func (s *ImmediateSendStrategy) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	n.SetSendTime()
	created, err := s.notifRepo.Create(ctx, n)
	if err != nil {
		return s.sender.Send(ctx, n)
	}

	if !errors.Is(err, errs.ErrDuplicateNotificationId) {
		// 非主键冲突，直接返回错误
		return domain.SendResp{}, fmt.Errorf("%w: failed to create notification", err)
	}

	found, err := s.notifRepo.GetByKey(ctx, created.BizId, created.BizKey)
	if err != nil {
		return domain.SendResp{}, fmt.Errorf("%w: failed to get notification", err)
	}

	if found.Status == domain.SendStatusSuccess {
		return domain.SendResp{
			Result: domain.SendResult{
				NotificationId: found.Id,
				Status:         found.Status,
			},
		}, nil
	}

	if found.Status != domain.SendStatusSending {
		return domain.SendResp{}, fmt.Errorf("%w", errs.ErrFailedToSendNotification)
	}

	found.Status = domain.SendStatusSending
	// 更新状态，获取乐观锁
	err = s.notifRepo.CompareAndSwapStatus(ctx, found)
	if err != nil {
		return domain.SendResp{}, fmt.Errorf("%w", err)
	}
	found.Version++
	return s.sender.Send(ctx, found)
}

func (s *ImmediateSendStrategy) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	if len(ns) == 0 {
		return domain.BatchSendResp{}, fmt.Errorf("%w: empty notification list", errs.ErrInvalidParam)
	}

	for i := range ns {
		ns[i].SetSendTime()
	}
	createdNs, err := s.notifRepo.BatchCreate(ctx, ns)
	if err != nil {
		return domain.BatchSendResp{}, fmt.Errorf("%w: failed to create notifications", err)
	}
	return s.sender.BatchSend(ctx, createdNs)
}

func NewImmediateSendStrategy(notifRepo repository.NotificationRepo, sender sender.Sender) *ImmediateSendStrategy {
	return &ImmediateSendStrategy{
		notifRepo: notifRepo,
		sender:    sender,
	}
}
