package domain

import (
	"time"

	"github.com/JrMarcco/jotify/internal/pkg/retry"
)

// TxNotifStatus 事务通知状态
type TxNotifStatus string

const (
	TxnStatusPrepare TxNotifStatus = "prepare"
	TxnStatusCommit  TxNotifStatus = "commit"
	TxnStatusCancel  TxNotifStatus = "cancel"
	TxnStatusFailed  TxNotifStatus = "failed"
)

func (tns TxNotifStatus) String() string {
	return string(tns)
}

// TxNotification 事务通知领域对象
type TxNotification struct {
	TxId   uint64
	BizId  uint64
	BizKey string

	Notification Notification
	Status       TxNotifStatus

	CheckedBackCnt  int32
	NextCheckBackAt int64
	CreateAt        int64
	UpdateAt        int64
}

func (tn *TxNotification) SetSendTime() {
	tn.Notification.SetSendTime()
}

func (tn *TxNotification) SetNextCheckAtAndStatus(txNotifConfig *TxNotifConfig) {
	if next, ok := tn.nextCheck(txNotifConfig); ok {
		tn.NextCheckBackAt = time.Now().Add(next).UnixMilli()
		return
	}

	tn.NextCheckBackAt = 0
	tn.Status = TxnStatusFailed

}

func (tn *TxNotification) nextCheck(txNotifConfig *TxNotifConfig) (time.Duration, bool) {
	if txNotifConfig == nil || txNotifConfig.RetryPolicy == nil {
		return 0, false
	}

	strategy, err := retry.NewRetryStrategy(*txNotifConfig.RetryPolicy)
	if err != nil {
		return 0, false
	}

	return strategy.NextWithRetried(tn.CheckedBackCnt)
}
