package domain

import "github.com/JrMarcco/jotify/internal/pkg/retry"

// BizConf 业务配置领域对象
type BizConf struct {
	Id           uint64
	OwnerId      uint64
	OwnerType    string
	ChannelConf  *ChannelConf
	TxNotifConf  *TxNotifConf
	RateLimit    int32
	QuotaConf    *QuotaConf
	CallbackConf *CallbackConf
	CreateAt     int64
	UpdateAt     int64
}

// ChannelConf 渠道配置领域对象
type ChannelConf struct {
	Channels    []ChannelItem `json:"channels"`
	RetryPolicy *retry.Config `json:"retry_policy"`
}

// ChannelItem 渠道项领域对象
type ChannelItem struct {
	Channel  string `json:"channel"`
	Priority int32  `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// TxNotifConf 事务消息配置领域对象
type TxNotifConf struct {
	ServiceName  string        `json:"service_name"`
	InitialDelay int32         `json:"initial_delay"`
	RetryPolicy  *retry.Config `json:"retry_policy"`
}

// CallbackConf 回调配置领域对象
type CallbackConf struct {
	ServiceName string        `json:"service_name"`
	RetryPolicy *retry.Config `json:"retry_policy"`
}

// QuotaConf 配额配置领域对象
type QuotaConf struct {
	Daily   *DailyQuotaConf   `json:"daily"`
	Monthly *MonthlyQuotaConf `json:"monthly"`
}

// DailyQuotaConf 日配额配置领域对象
type DailyQuotaConf struct {
	SMS   int32 `json:"sms"`
	Email int32 `json:"email"`
}

// MonthlyQuotaConf 月度配额领域对象
type MonthlyQuotaConf struct {
	SMS   int32 `json:"sms"`
	Email int32 `json:"email"`
}
