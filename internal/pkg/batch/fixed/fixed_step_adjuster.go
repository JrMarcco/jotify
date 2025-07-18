package fixed

import (
	"context"
	"time"

	"github.com/JrMarcco/jotify/internal/pkg/batch"
)

var _ batch.Adjuster = (*Adjuster)(nil)

type Adjuster struct {
	currSize   uint64 // 当前批次大小
	minSize    uint64 // 最小批次大小
	maxSize    uint64 // 最大批次大小
	adjustStep uint64 // 调整步长2

	lastAdjustTime time.Time // 上次调整时间

	minAdjustInterval time.Duration // 最小调整间隔
	fastThreshold     time.Duration // 增加步长的阈值
	slowThreshold     time.Duration // 减少步长的阈值
}

func (a *Adjuster) Adjust(_ context.Context, respTime time.Duration) (uint64, error) {
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

func NewAdjuster(
	initSize, minSize, maxSize, adjustStep uint64,
	minAdjustInterval, fastThreshold, slowThreshold time.Duration,
) *Adjuster {
	if initSize < minSize {
		initSize = minSize
	}
	if initSize > maxSize {
		initSize = maxSize
	}

	return &Adjuster{
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
