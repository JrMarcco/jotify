package domain

import (
	"fmt"
	"time"

	"github.com/JrMarcco/jotify/internal/errs"
)

// SendStrategy 发送策略
type SendStrategy string

func (s SendStrategy) String() string {
	return string(s)
}

const (
	SendStrategyImmediate  SendStrategy = "immediate"
	SendStrategyDelayed    SendStrategy = "delayed"
	SendStrategyScheduled  SendStrategy = "scheduled"
	SendStrategyTimeWindow SendStrategy = "time_window"
	SendStrategyDeadline   SendStrategy = "deadline"
)

// SendStrategyConfig 发送策略配置
type SendStrategyConfig struct {
	Type       SendStrategy  `json:"type"`
	Delay      time.Duration `json:"delay"`
	ScheduleAt time.Time     `json:"schedule_at"`
	Start      time.Time     `json:"start"`
	End        time.Time     `json:"end"`
	Deadline   time.Time     `json:"deadline"`
}

// Validate 校验发送策略配置
func (c SendStrategyConfig) Validate() error {
	switch c.Type {
	case SendStrategyImmediate:
		return nil
	case SendStrategyDelayed:
		if c.Delay <= 0 {
			return fmt.Errorf("%w: delay time should be greater than 0", errs.ErrInvalidParam)
		}
	case SendStrategyScheduled:
		if c.ScheduleAt.IsZero() || c.ScheduleAt.Before(time.Now()) {
			return fmt.Errorf("%w: schedule_at should not be zero or before now", errs.ErrInvalidParam)
		}
	case SendStrategyTimeWindow:
		if c.Start.IsZero() || c.Start.After(c.End) {
			return fmt.Errorf("%w: start and end time should not be zero and start should be before end", errs.ErrInvalidParam)
		}
	case SendStrategyDeadline:
		if c.Deadline.IsZero() || c.Deadline.Before(time.Now()) {
			return fmt.Errorf("%w: deadline should not be zero or before now", errs.ErrInvalidParam)
		}
	default:
		return fmt.Errorf("%w: unknown strategy", errs.ErrInvalidParam)
	}

	return nil
}

// CalcTimeWindow 计算发送时间窗口
func (c SendStrategyConfig) CalcTimeWindow() (start, end time.Time) {
	switch c.Type {
	case SendStrategyImmediate:
		// immediately send
		now := time.Now()
		const defaultEndDuration = 30 * time.Minute
		return now, now.Add(defaultEndDuration)
	case SendStrategyDelayed:
		now := time.Now()
		return now, now.Add(c.Delay)
	case SendStrategyDeadline:
		now := time.Now()
		return now, c.Deadline
	case SendStrategyTimeWindow:
		return c.Start, c.End
	case SendStrategyScheduled:
		const scheduledTimeTolerance = 3 * time.Second
		return c.Start.Add(-scheduledTimeTolerance), c.Deadline
	default:
		now := time.Now()
		return now, now
	}
}

// SendResult 发送结果
type SendResult struct {
	NotificationId uint64     // notification 实体的 id
	Status         SendStatus // 发送状态
}

// SendResp 发送请求的响应
type SendResp struct {
	Result SendResult
}

// BatchSendResp 批量发送请求的响应
type BatchSendResp struct {
	Results []SendResult
}

// BatchAsyncSendResp 批量异步发送请求的响应
type BatchAsyncSendResp struct {
	NotificationIds []uint64
}
