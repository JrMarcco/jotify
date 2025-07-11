package batch

import (
	"context"
	"time"
)

//go:generate mockgen -source=./types.go -destination=./mock/adjuster.mock.go -package=batchmock -typed Adjuster

// Adjuster 批任务批次大小调整器，根据响应时间来调整
type Adjuster interface {
	// Adjust 批次大小调整
	Adjust(ctx context.Context, respTime time.Duration) (int, error)
}
