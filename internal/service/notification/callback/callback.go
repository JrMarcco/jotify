package callback

import (
	"context"
	"fmt"
	"time"

	"github.com/JrMarcco/easy-kit/xsync"
	clientv1 "github.com/JrMarcco/jotify-api/api/client/v1"
	notificationv1 "github.com/JrMarcco/jotify-api/api/notification/v1"
	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	grpcpkg "github.com/JrMarcco/jotify/internal/pkg/grpc"
	"github.com/JrMarcco/jotify/internal/pkg/retry"
	"github.com/JrMarcco/jotify/internal/repository"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Service interface {
	Send(ctx context.Context, startTime int64, batchSize int) error
	SendByNotification(ctx context.Context, n domain.Notification) error
	SendByNotifications(ctx context.Context, ns []domain.Notification) error
}

var _ Service = (*DefaultService)(nil)

type DefaultService struct {
	clients *grpcpkg.Clients[clientv1.CallbackServiceClient]

	bizConfRepo     repository.BizConfRepo
	callbackLogRepo repository.CallbackLogRepo

	confMap xsync.Map[uint64, *domain.CallbackConf]
	logger  *zap.Logger
}

func (d *DefaultService) Send(ctx context.Context, startTime int64, batchSize int) error {
	nextStartId := uint64(0)
	for {
		logs, newId, err := d.callbackLogRepo.Find(ctx, startTime, nextStartId, batchSize)
		if err != nil {
			d.logger.Error(
				"[jotify] failed to find callback logs",
				zap.Int64("start_time", startTime),
				zap.Uint64("next_start_id", nextStartId),
				zap.Int("batch_size", batchSize),
				zap.Error(err),
			)
			return err
		}

		if len(logs) == 0 {
			break
		}

		// 处理当前批次回调
		err = d.sendAndUpdateLogs(ctx, logs)
		if err != nil {
			return err
		}
		nextStartId = newId
	}
	return nil
}

// sendAndUpdateLogs 发送回调并更新 callback log
func (d *DefaultService) sendAndUpdateLogs(ctx context.Context, logs []domain.CallbackLog) error {
	updateParams := make([]domain.CallbackLog, 0, len(logs))
	for i := range logs {
		changed, err := d.sendAndSetChangeFields(ctx, logs[i])
		if err != nil {
			d.logger.Error(
				"[jotify] failed to send callback log",
				zap.Uint64("callback.id", logs[i].Notification.Id),
				zap.Error(err),
			)
			continue
		}
		if changed {
			// 需要更新，添加 callback log 到待更新参数列表
			updateParams = append(updateParams, logs[i])
		}
	}
	return d.callbackLogRepo.Update(ctx, updateParams)
}

// sendAndSetChangeFields 实际发送回调请求并判断是否需要更新 callback log 信息
// 返回 true 表示需要更新信息
func (d *DefaultService) sendAndSetChangeFields(ctx context.Context, log domain.CallbackLog) (bool, error) {
	// 实际发送回调
	resp, err := d.sendCallback(ctx, log.Notification)
	if err != nil {
		return false, err
	}

	if resp.Success {
		log.Status = domain.CallbackStatusSuccess
		return true, nil
	}

	// 回调请求失败，重试
	conf, err := d.getConf(ctx, log.Notification.BizId)
	if err != nil {
		// 业务方必须相应的业务配置，理论上个分支不会进来
		d.logger.Error("[jotify] failed to get biz conf", zap.Uint64("biz_id", log.Notification.BizId), zap.Error(err))
		return false, err
	}
	strategy, err := retry.NewRetryStrategy(*conf.RetryPolicy)
	if err != nil {
		// 同上，理论上不会进来这个分支
		d.logger.Error("[jotify] failed to create retry strategy", zap.Error(err))
		return false, err
	}

	interval, ok := strategy.NextWithRetried(log.RetriedTimes)
	if ok {
		log.NextRetryAt = time.Now().Add(interval).Unix()
		log.RetriedTimes++
		return true, nil
	}
	// 达到最大重试次数，直接更新为失败
	log.Status = domain.CallbackStatusFailure
	return true, nil
}

func (d *DefaultService) SendByNotification(ctx context.Context, n domain.Notification) error {
	logs, err := d.callbackLogRepo.FindByNotificationIds(ctx, []uint64{n.Id})
	if err != nil {
		return err
	}
	return d.sendAndUpdateLogs(ctx, logs)
}

func (d *DefaultService) SendByNotifications(ctx context.Context, ns []domain.Notification) error {
	nIds := make([]uint64, 0, len(ns))
	m := make(map[uint64]domain.Notification, len(ns))

	for i := range ns {
		nIds = append(nIds, ns[i].Id)
		m[ns[i].Id] = ns[i]
	}

	logs, err := d.callbackLogRepo.FindByNotificationIds(ctx, nIds)
	if err != nil {
		return err
	}

	if len(logs) == len(ns) {
		// 当前所有的通知都已经存在对应的回调日志
		return d.sendAndUpdateLogs(ctx, logs)
	}
	for i := range ns {
		delete(m, ns[i].Id)
	}

	if len(logs) != 0 {
		// 部分消息存在回调日志
		err = d.callbackLogRepo.Update(ctx, logs)
	}

	for _, val := range m {
		_, err = d.sendCallback(ctx, val)
	}
	return err
}

func (d *DefaultService) sendCallback(ctx context.Context, n domain.Notification) (*clientv1.SendResultNotifyResponse, error) {
	conf, err := d.getConf(ctx, n.BizId)
	if err != nil {
		d.logger.Warn("[jotify] failed to get biz conf", zap.Uint64("biz_id", n.BizId), zap.Error(err))
		return nil, err
	}

	if conf == nil {
		return nil, fmt.Errorf("%w", errs.ErrBizConfNotFound)
	}
	return d.clients.Get(conf.ServiceName).SendResultNotify(ctx, d.buildNotifyReq(n))
}

func (d *DefaultService) getConf(ctx context.Context, bizId uint64) (*domain.CallbackConf, error) {
	conf, ok := d.confMap.Load(bizId)
	if ok {
		return conf, nil
	}

	bizConf, err := d.bizConfRepo.GetById(ctx, bizId)
	if err != nil {
		return nil, err
	}

	if bizConf.CallbackConf != nil {
		d.confMap.Store(bizId, bizConf.CallbackConf)
	}
	return bizConf.CallbackConf, nil
}

func (d *DefaultService) buildNotifyReq(n domain.Notification) *clientv1.SendResultNotifyRequest {
	tplParams := make(map[string]string)
	if n.Template.Params != nil {
		tplParams = n.Template.Params
	}
	return &clientv1.SendResultNotifyRequest{
		NotificationId: n.Id,
		OriRequest: &notificationv1.SendRequest{
			Notification: &notificationv1.Notification{
				BizKey:    n.BizKey,
				Receivers: n.Receivers,
				Channel:   d.getChannel(n),
				TplId:     fmt.Sprintf("%d", n.Template.Id),
				TplParams: tplParams,
			},
		},
		Result: &notificationv1.SendResult{
			NotificationId: n.Id,
			Status:         d.getStatus(n),
		},
	}
}

func (d *DefaultService) getChannel(n domain.Notification) notificationv1.Channel {
	channel := notificationv1.Channel_CHANNEL_UNSPECIFIED
	switch n.Channel {
	case domain.ChannelSMS:
		channel = notificationv1.Channel_SMS
	case domain.ChannelEmail:
		channel = notificationv1.Channel_EMAIL
	case domain.ChannelApp:
		channel = notificationv1.Channel_IN_APP
	}
	return channel
}

func (d *DefaultService) getStatus(n domain.Notification) notificationv1.SendStatus {
	status := notificationv1.SendStatus_STATUS_UNSPECIFIED
	switch n.Status {
	case domain.SendStatusSuccess:
		status = notificationv1.SendStatus_SUCCESS
	case domain.SendStatusFailure:
		status = notificationv1.SendStatus_FAILURE
	case domain.SendStatusPrepare:
		status = notificationv1.SendStatus_PREPARE
	case domain.SendStatusPending:
		status = notificationv1.SendStatus_PENDING
	case domain.SendStatusCancel:
		status = notificationv1.SendStatus_CANCEL
	case domain.SendStatusSending:
		// domain.SendStatusSending -> notification v1.SendStatus_STATUS_UNSPECIFIED
	}
	return status
}

func NewDefaultService(
	etcdClient *clientv3.Client,
	bizConfRepo repository.BizConfRepo,
	callbackLogRepo repository.CallbackLogRepo,
	logger *zap.Logger,
) *DefaultService {
	clients := grpcpkg.NewClients(etcdClient, func(conn *grpc.ClientConn) clientv1.CallbackServiceClient {
		return clientv1.NewCallbackServiceClient(conn)
	})
	return &DefaultService{
		clients:         clients,
		bizConfRepo:     bizConfRepo,
		callbackLogRepo: callbackLogRepo,
		logger:          logger,
	}
}
