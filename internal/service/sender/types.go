package sender

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/service/channel"
	"go.uber.org/zap"
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

	logger *zap.Logger
}

func (ds *DefaultSender) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	//TODO implement me
	panic("implement me")
}

func (ds *DefaultSender) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	//TODO implement me
	panic("implement me")
}

func NewDefaultSender() *DefaultSender {
	return &DefaultSender{}
}
