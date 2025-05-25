package domain

// CallbackStatus 回调状态
type CallbackStatus string

const (
	CallbackStatusInit    CallbackStatus = "init"
	CallbackStatusPending CallbackStatus = "pending"
	CallbackStatusSucceed CallbackStatus = "succeed"
	CallbackStatusFailed  CallbackStatus = "failed"
)

func (s CallbackStatus) String() string {
	return string(s)
}

// CallbackLog 回调日志领域对象
type CallbackLog struct {
	Id           uint64
	Notification Notification
	RetriedTimes int32
	NextRetryAt  int64
	Status       CallbackStatus
}
