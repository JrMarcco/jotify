package domain

// CallbackLogStatus 回调状态
type CallbackLogStatus string

const (
	CallbackStatusInit    CallbackLogStatus = "init"
	CallbackStatusPending CallbackLogStatus = "pending"
	CallbackStatusSuccess CallbackLogStatus = "success"
	CallbackStatusFailure CallbackLogStatus = "failure"
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
