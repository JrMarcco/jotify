package domain

// BizConfig 业务配置领域对象
type BizConfig struct {
	Id             uint64
	OwnerId        uint64
	OwnerType      string
	ChannelConfig  *ChannelConfig
	TxNotifConfig  *TxNotifConfig
	RateLimit      int32
	QuotaConfig    *QuotaConfig
	CallbackConfig *CallbackConfig
	CreateAt       int64
	UpdateAt       int64
}

// ChannelConfig 渠道配置领域对象
type ChannelConfig struct {
	Channels    []ChannelItem `json:"channels"`
	RetryPolicy *retry.Config `json:"retry_policy"`
}

// ChannelItem 渠道项领域对象
type ChannelItem struct {
	Channel  string `json:"channel"`
	Priority int32  `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// TxNotifConfig 事务消息配置领域对象
type TxNotifConfig struct {
	ServiceName  string        `json:"service_name"`
	InitialDelay int32         `json:"initial_delay"`
	RetryPolicy  *retry.Config `json:"retry_policy"`
}

// CallbackConfig 回调配置领域对象
type CallbackConfig struct {
	ServiceName string        `json:"service_name"`
	RetryPolicy *retry.Config `json:"retry_policy"`
}

// QuotaConfig 配额配置领域对象
type QuotaConfig struct {
	Daily   *DailyQuotaConfig   `json:"daily"`
	Monthly *MonthlyQuotaConfig `json:"monthly"`
}

// DailyQuotaConfig 日配额配置领域对象
type DailyQuotaConfig struct {
	SMS   int32 `json:"sms"`
	Email int32 `json:"email"`
}

// MonthlyQuotaConfig 月度配额领域对象
type MonthlyQuotaConfig struct {
	SMS   int32 `json:"sms"`
	Email int32 `json:"email"`
}
