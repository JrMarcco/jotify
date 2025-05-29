package channel

import (
	"context"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
)

//go:generate mockgen -source=./types.go -destination=./mock/channel.mock.go -package=channelmock -typed Channel

// Channel 发送渠道接口。
type Channel interface {
	Send(ctx context.Context, n domain.Notification) (domain.SendResp, error)
}

var _ Channel = (*Dispatcher)(nil)

// Dispatcher 渠道分发器，作为对外同意入口。
type Dispatcher struct {
	channels map[domain.Channel]Channel
}

func (d *Dispatcher) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	if channel, ok := d.channels[n.Channel]; ok {
		return channel.Send(ctx, n)
	}
	return domain.SendResp{}, fmt.Errorf("%w", errs.ErrInvalidChannel)
}

func NewDispatcher(channels map[domain.Channel]Channel) *Dispatcher {
	return &Dispatcher{
		channels: channels,
	}
}
