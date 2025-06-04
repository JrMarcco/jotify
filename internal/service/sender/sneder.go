package sender

import (
	"context"
	"fmt"
	"sync"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/service/channel"
	"github.com/JrMarcco/jotify/internal/service/notification"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source=./types.go -destination=./mock/provider.mock.go -package=sendermock -typed Sender

// Sender 消息发送器接口（实际的发送逻辑）
//
// 相比之下 SendService 处理的是发送前的准备工作。
// 例如 Notification 记录入库，配额管理等。
type Sender interface {
	Send(ctx context.Context, n domain.Notification) (domain.SendResp, error)
	BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error)
}

var _ Sender = (*DefaultSender)(nil)

// DefaultSender 默认发送器接口实现
type DefaultSender struct {
	channel channel.Channel

	notifRepo   repository.NotificationRepo
	bizConfRepo repository.BizConfRepo

	callbackSvc notification.CallbackService

	logger *zap.Logger
}

func (ds *DefaultSender) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	res := domain.SendResult{NotificationId: n.Id}

	_, err := ds.channel.Send(ctx, n)
	if err != nil {
		ds.logger.Error("[jotify] send notification error", zap.Error(err))
		res.Status = domain.SendStatusFailure
		n.Status = domain.SendStatusFailure

		// 发送失败，把 quota 加回去
		err = ds.notifRepo.MarkFailure(ctx, n)
	} else {
		res.Status = domain.SendStatusSuccess
		n.Status = domain.SendStatusSuccess
		err = ds.notifRepo.MarkSuccess(ctx, n)
	}

	if err != nil {
		return domain.SendResp{}, err
	}

	// 处理回调通知发送结果
	_ = ds.callbackSvc.SendByNotification(ctx, n)

	return domain.SendResp{Result: res}, nil
}

func (ds *DefaultSender) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	if len(ns) == 0 {
		return domain.BatchSendResp{}, nil
	}

	var successMu, failureMu sync.Mutex
	var success, failure []domain.SendResult
	var eg errgroup.Group
	for i := range ns {
		n := ns[i]
		// TODO: 这里可以考虑做 task pool 来控制 goroutine 数量
		eg.Go(func() error {
			_, err := ds.channel.Send(ctx, n)
			if err != nil {
				res := domain.SendResult{
					NotificationId: n.Id,
					Status:         domain.SendStatusFailure,
				}
				failureMu.Lock()
				failure = append(failure, res)
				failureMu.Unlock()
				return nil
			}

			res := domain.SendResult{
				NotificationId: n.Id,
				Status:         domain.SendStatusSuccess,
			}
			successMu.Lock()
			success = append(success, res)
			successMu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()

	allIds := make([]uint64, 0, len(success)+len(failure))
	for _, res := range success {
		allIds = append(allIds, res.NotificationId)
	}
	for _, res := range failure {
		allIds = append(allIds, res.NotificationId)
	}

	m, err := ds.notifRepo.GetMapByIds(ctx, allIds)
	if err != nil {
		ds.logger.Error(
			"[jotify] failed to batch get notifications",
			zap.Any("ids", allIds),
			zap.Error(err),
		)
		return domain.BatchSendResp{}, fmt.Errorf("[jotify] failed to batch get notifications: %w", err)
	}

	successNs := ds.getNsWithStatus(success, m)
	failureNs := ds.getNsWithStatus(failure, m)

	err = ds.batchUpdateStatus(ctx, successNs, failureNs)
	if err != nil {
		return domain.BatchSendResp{}, fmt.Errorf("[jotify] failed to batch update notification status: %w", err)
	}

	_ = ds.callbackSvc.SendByNotifications(ctx, append(successNs, failureNs...))
	return domain.BatchSendResp{
		Results: append(success, failure...),
	}, nil
}

func (ds *DefaultSender) getNsWithStatus(results []domain.SendResult, nMap map[uint64]domain.Notification) []domain.Notification {
	ns := make([]domain.Notification, 0, len(results))
	for _, res := range results {
		if n, ok := nMap[res.NotificationId]; ok {
			n.Status = res.Status
			ns = append(ns, n)
		}
	}
	return ns
}

func (ds *DefaultSender) batchUpdateStatus(ctx context.Context, successNs, failureNs []domain.Notification) error {
	if len(successNs) > 0 || len(failureNs) > 0 {
	}
	return nil
}

func NewDefaultSender(
	channel channel.Channel,
	notifRepo repository.NotificationRepo,
	bizConfRepo repository.BizConfRepo,
	callbackSvc notification.CallbackService,
	logger *zap.Logger,
) *DefaultSender {
	return &DefaultSender{
		channel:     channel,
		notifRepo:   notifRepo,
		bizConfRepo: bizConfRepo,
		callbackSvc: callbackSvc,
		logger:      logger,
	}
}
