package batch

import (
	"context"
	"time"
)

var _ Adjuster = (*FixedStepAdjuster)(nil)

type FixedStepAdjuster struct {
	currSize   int // 当前批次大小
	minSize    int // 最小批次大小
	maxSize    int // 最大批次大小
	adjustStep int // 调整步长2

	lastAdjustTime time.Time // 上次调整时间

	minAdjustInterval time.Duration // 最小调整间隔
	fastThreshold     time.Duration // 增加步长的阈值
	slowThreshold     time.Duration // 减少步长的阈值
}

func (a *FixedStepAdjuster) Adjust(_ context.Context, respTime time.Duration) (int, error) {
	if !a.lastAdjustTime.IsZero() && time.Since(a.lastAdjustTime) < a.minAdjustInterval {
		return a.currSize, nil
	}

	// 响应较快，增加步长
	if respTime < a.fastThreshold {
		if a.currSize < a.maxSize {
			a.currSize = min(a.currSize+a.adjustStep, a.maxSize)
			a.lastAdjustTime = time.Now()
		}
		return a.currSize, nil
	}

	// 响应较慢，减少步长
	if respTime > a.slowThreshold {
		if a.currSize > a.minSize {
			a.currSize = max(a.currSize-a.adjustStep, a.minSize)
			a.lastAdjustTime = time.Now()
		}
	}
	return a.currSize, nil
}

func NewFixedStepAdjuster(
	initSize, minSize, maxSize, adjustStep int,
	minAdjustInterval, fastThreshold, slowThreshold time.Duration,
) *FixedStepAdjuster {
	if initSize < minSize {
		initSize = minSize
	}
	if initSize > maxSize {
		initSize = maxSize
	}

	return &FixedStepAdjuster{
		currSize:          initSize,
		minSize:           minSize,
		maxSize:           maxSize,
		adjustStep:        adjustStep,
		lastAdjustTime:    time.Time{},
		minAdjustInterval: minAdjustInterval,
		fastThreshold:     fastThreshold,
		slowThreshold:     slowThreshold,
	}
}
