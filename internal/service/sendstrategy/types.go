package sendstrategy

import (
	"context"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
)

// SendStrategy 消息发送策略。
//
// 目前两种实现：
// DefaultSendStrategy 		默认，使用异步发送（存库等待后续调度执行实际发送）
// ImmediateSendStrategy	立即发送，即同步
//
//go:generate mockegn -source=./types.go -destination=./mock/send_strategy.mock.go -package=sendstrategymock -type=SendStrategy
type SendStrategy interface {
	Send(ctx context.Context, n domain.Notification) (domain.SendResp, error)
	BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error)
}

var _ SendStrategy = (*Dispatcher)(nil)

// Dispatcher is a strategy dispatcher that chooses the appropriate strategy based on the notification's strategy configuration.
type Dispatcher struct {
	defaultStrategy   *DefaultSendStrategy
	immediateStrategy *ImmediateSendStrategy
}

func (d *Dispatcher) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	return d.chooseStrategy(n).Send(ctx, n)
}

func (d *Dispatcher) BatchSend(ctx context.Context, ns []domain.Notification) (domain.BatchSendResp, error) {
	if len(ns) == 0 {
		return domain.BatchSendResp{}, fmt.Errorf("%w: no notifications to send", errs.ErrInvalidParam)
	}

	return d.chooseStrategy(ns[0]).BatchSend(ctx, ns)
}

func (d *Dispatcher) chooseStrategy(n domain.Notification) SendStrategy {
	if n.StrategyConfig.Type == domain.SendStrategyImmediate {
		return d.immediateStrategy
	}

	return d.defaultStrategy
}

func NewDispatcher(defaultStrategy *DefaultSendStrategy, immediateStrategy *ImmediateSendStrategy) *Dispatcher {
	return &Dispatcher{
		defaultStrategy:   defaultStrategy,
		immediateStrategy: immediateStrategy,
	}
}
