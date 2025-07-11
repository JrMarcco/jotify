package schedule

import "context"

// NotifScheduler 通知调度服务接口
type NotifScheduler interface {
	// Start 启动调度
	Start(ctx context.Context) error
}
