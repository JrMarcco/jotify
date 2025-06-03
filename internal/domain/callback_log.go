package domain

// CallbackLogStatus 回调状态
type CallbackLogStatus string

const (
	CallbackStatusInit    CallbackLogStatus = "init"
	CallbackStatusPending CallbackLogStatus = "pending"
	CallbackStatusSucceed CallbackLogStatus = "succeed"
	CallbackStatusFailed  CallbackLogStatus = "failed"
)

func (s CallbackLogStatus) String() string {
	return string(s)
}

// CallbackLog 回调日志领域对象
type CallbackLog struct {
	Notification Notification
	RetriedTimes int32
	NextRetryAt  int64
	Status       CallbackLogStatus
}
