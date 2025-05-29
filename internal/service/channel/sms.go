package channel

import (
	"context"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/service/provider"
)

var _ Channel = (*baseChannel)(nil)

type baseChannel struct {
	sb provider.SelectorBuilder
}

func (bc *baseChannel) Send(ctx context.Context, n domain.Notification) (domain.SendResp, error) {
	selector, err := bc.sb.Build()
	if err != nil {
		return domain.SendResp{}, fmt.Errorf("%w: %w", errs.ErrFailedToSendNotification, err)
	}

	for {
		p, selectErr := selector.Next(ctx, n)
		if err != nil {
			return domain.SendResp{}, fmt.Errorf("%w: %w", errs.ErrFailedToSendNotification, selectErr)
		}

		resp, sendErr := p.Send(ctx, n)
		if sendErr != nil {
			// 执行发送异常，则直接循环调用 selector.Next 获取下一个供应商来发送
			continue
		}
		return resp, nil
	}
}

var _ Channel = (*SmsChannel)(nil)

type SmsChannel struct {
	baseChannel
}

func NewSmsChannel(sb provider.SelectorBuilder) *SmsChannel {
	return &SmsChannel{
		baseChannel: baseChannel{
			sb: sb,
		},
	}
}
