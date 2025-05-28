package dao

type CallbackLog struct {
	NotificationId uint64
	RetriedTimes   int32
	NextRetryAt    int64
	Status         string
	CreatedAt      int64
	UpdatedAt      int64
}
