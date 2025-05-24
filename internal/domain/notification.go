package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	notificationv1 "github.com/JrMarcco/jotify-api/api/notification/v1"
	"github.com/JrMarcco/jotify/internal/errs"
)

type SendStatus string

const (
	SendStatusPrepare  SendStatus = "prepare"
	SendStatusCanceled SendStatus = "canceled"
	SendStatusPending  SendStatus = "pending"
	SendStatusSending  SendStatus = "sending"
	SendStatusSuccess  SendStatus = "success"
	SendStatusFailed   SendStatus = "failed"
)

func (s SendStatus) String() string {
	return string(s)
}

// Template 消息模板领域对象
type Template struct {
	Id        uint64            `json:"id"`
	VersionId uint64            `json:"version_id"`
	Params    map[string]string `json:"params"`
}

// Notification 消息领域对象
type Notification struct {
	Id             uint64           `json:"id"`
	BizId          uint64           `json:"biz_id"`
	BizKey         string           `json:"biz_key"`
	Receivers      []string         `json:"receivers"`
	Channel        Channel          `json:"channel"`
	Template       Template         `json:"template"`
	Status         SendStatus       `json:"status"`
	ScheduledStart time.Time        `json:"scheduled_start"`
	ScheduledEnd   time.Time        `json:"scheduled_end"`
	Version        int32            `json:"version"`
	StrategyConfig SendStrategyConf `json:"strategy_config"`
}

func (n *Notification) Validate() error {
	if n.BizId <= 0 {
		return fmt.Errorf("%w: biz id should not be negative or zero", errs.ErrInvalidParam)
	}

	if n.BizKey == "" {
		return fmt.Errorf("%w: biz key should not be empty", errs.ErrInvalidParam)
	}

	if len(n.Receivers) == 0 {
		return fmt.Errorf("%w: receivers should not be empty", errs.ErrInvalidParam)
	}

	if !n.Channel.Validate() {
		return fmt.Errorf("%w: invalid channel", errs.ErrInvalidParam)
	}

	if n.Template.Id <= 0 {
		return fmt.Errorf("%w: template id should not be negative or zero", errs.ErrInvalidParam)
	}

	if n.Template.VersionId <= 0 {
		return fmt.Errorf("%w: template version id should not be negative or zero", errs.ErrInvalidParam)
	}

	if len(n.Template.Params) == 0 {
		return fmt.Errorf("%w: template params should not be empty", errs.ErrInvalidParam)
	}

	if err := n.StrategyConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func (n *Notification) ValidatedBizId() error {
	if n.BizId <= 0 {
		return fmt.Errorf("%w: BizId = %q", errs.ErrInvalidParam, n.BizId)
	}
	return nil
}

// SetSendTime 设置消息发送时间窗口
//
// 计算消息发送时间窗口并设置窗口开始时间、窗口结束时间。
// 通过调度任务将消息在时间窗口内发出。
func (n *Notification) SetSendTime() {
	start, end := n.StrategyConfig.CalcTimeWindow()
	n.ScheduledStart = start
	n.ScheduledEnd = end
}

func (n *Notification) IsImmediate() bool {
	return n.StrategyConfig.Type == SendStrategyImmediate
}

// ReplaceAsyncImmediate 将立即发送消息转为异步发送消息，Deadline 为一分钟。
func (n *Notification) ReplaceAsyncImmediate() {
	if n.IsImmediate() {
		n.StrategyConfig.Deadline = time.Now().Add(time.Minute)
		n.StrategyConfig.Type = SendStrategyDeadline
	}
}

func (n *Notification) MarshalReceivers() (string, error) {
	return n.marshal(n.Receivers)
}

func (n *Notification) MarshalTemplateParams() (string, error) {
	return n.marshal(n.Template.Params)
}

func (n *Notification) marshal(val any) (string, error) {
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func NotificationFromApi(n *notificationv1.Notification) (Notification, error) {
	if n == nil {
		return Notification{}, fmt.Errorf("%w: notification is nil", errs.ErrInvalidParam)
	}

	tplId, err := strconv.ParseUint(n.TempId, 10, 64)
	if err != nil {
		return Notification{}, fmt.Errorf("%w: parse template id failed: %w", errs.ErrInvalidParam, err)
	}

	channel, err := getDomainChannel(n)
	if err != nil {
		return Notification{}, err
	}

	strategyConfig, err := getDomainStrategyConfig(n)
	if err != nil {
		return Notification{}, err
	}

	return Notification{
		BizKey:    n.Key,
		Receivers: n.Receivers,
		Channel:   channel,
		Template: Template{
			Id:     tplId,
			Params: n.TempParams,
		},
		StrategyConfig: strategyConfig,
	}, nil
}

func getDomainChannel(n *notificationv1.Notification) (Channel, error) {
	switch n.Channel {
	case notificationv1.Channel_SMS:
		return ChannelSMS, nil
	case notificationv1.Channel_EMAIL:
		return ChannelEmail, nil
	case notificationv1.Channel_IN_APP:
		return ChannelApp, nil
	default:
		return "", fmt.Errorf("%w", errs.ErrInvalidChannel)
	}
}

func getDomainStrategyConfig(n *notificationv1.Notification) (SendStrategyConf, error) {
	var strategy SendStrategy
	var delaySeconds int64
	var scheduleAt time.Time
	var startTime int64
	var endTime int64
	var deadline time.Time

	if n.Strategy != nil {
		switch st := n.Strategy.StrategyType.(type) {
		case *notificationv1.SendStrategy_Immediate:
			strategy = SendStrategyImmediate
		case *notificationv1.SendStrategy_Delayed:
			if st.Delayed != nil && st.Delayed.DelaySeconds > 0 {
				strategy = SendStrategyDelayed
				delaySeconds = st.Delayed.DelaySeconds
				break
			}
			return SendStrategyConf{}, fmt.Errorf("%w", errs.ErrInvalidSendStrategy)
		case *notificationv1.SendStrategy_Scheduled:
			if st.Scheduled != nil && st.Scheduled.SendTime != nil {
				strategy = SendStrategyScheduled
				scheduleAt = st.Scheduled.SendTime.AsTime()
				break
			}
			return SendStrategyConf{}, fmt.Errorf("%w", errs.ErrInvalidSendStrategy)
		case *notificationv1.SendStrategy_TimeWindow:
			if st.TimeWindow != nil {
				strategy = SendStrategyTimeWindow
				startTime = st.TimeWindow.StartTimeMillis
				endTime = st.TimeWindow.EndTimeMillis
				break
			}
			return SendStrategyConf{}, fmt.Errorf("%w", errs.ErrInvalidSendStrategy)
		case *notificationv1.SendStrategy_Deadline:
			if st.Deadline != nil {
				strategy = SendStrategyDeadline
				deadline = st.Deadline.Deadline.AsTime()
				break
			}
			return SendStrategyConf{}, fmt.Errorf("%w", errs.ErrInvalidSendStrategy)
		default:
			return SendStrategyConf{}, fmt.Errorf("%w", errs.ErrInvalidSendStrategy)
		}
	}

	return SendStrategyConf{
		Type:       strategy,
		Delay:      time.Duration(delaySeconds) * time.Second,
		ScheduleAt: scheduleAt,
		Start:      time.Unix(startTime, 0),
		End:        time.Unix(endTime, 0),
		Deadline:   deadline,
	}, nil
}
