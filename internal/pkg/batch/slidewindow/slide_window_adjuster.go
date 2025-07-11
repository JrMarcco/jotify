package slidewindow

import (
	"context"
	"sync"
	"time"

	"github.com/JrMarcco/jotify/internal/pkg/batch"
	"github.com/JrMarcco/jotify/internal/pkg/ringbuffer"
)

var _ batch.Adjuster = (*Adjuster)(nil)

// Adjuster 基于滑动窗口计算平均响应时间调整批任务的批次大小。
// 使用 ring buffer 来实现滑动窗口
type Adjuster struct {
	mu sync.RWMutex

	currSize   uint64 // 当前批次大小
	minSize    uint64 // 最小批次大小
	maxSize    uint64 // 最大批次大小
	adjustStep uint64 // 调整步长

	lastAdjustTime time.Time // 上次调整时间

	buffer            *ringbuffer.TimeDurationRingBuffer // 滑动窗口
	minAdjustInterval time.Duration                      // 最小调整间隔
}

func (a *Adjuster) Adjust(_ context.Context, respTime time.Duration) (uint64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.buffer.Add(respTime)

	// 至少要收集窗口才能开始调整
	if a.buffer.Count() < a.buffer.Size() {
		return a.currSize, nil
	}

	if !a.lastAdjustTime.IsZero() && time.Since(a.lastAdjustTime) < a.minAdjustInterval {
		return a.currSize, nil
	}

	avg := a.buffer.Avg()
	// 响应时间小于窗口平均时间，增加批次大小
	if respTime < avg {
		if a.currSize < a.maxSize {
			a.currSize = min(a.currSize+a.adjustStep, a.maxSize)
			a.lastAdjustTime = time.Now()
		}
		return a.currSize, nil
	}

	// 响应时间大于窗口平郡时间，减少批次大小
	if respTime > avg {
		if a.currSize > a.minSize {
			a.currSize = max(a.currSize-a.adjustStep, a.minSize)
			a.lastAdjustTime = time.Now()
		}
	}
	return a.currSize, nil
}

func NewAdjuster(
	bufferSize int, initSize, minSize, maxSize, adjustStep uint64, minAdjustInterval time.Duration,
) (*Adjuster, error) {
	if initSize < minSize {
		initSize = minSize
	}
	if initSize > maxSize {
		initSize = maxSize
	}

	buffer, err := ringbuffer.NewTimeDurationRingBuffer(bufferSize)
	if err != nil {
		return nil, err
	}

	return &Adjuster{
		currSize:          initSize,
		minSize:           minSize,
		maxSize:           maxSize,
		adjustStep:        adjustStep,
		lastAdjustTime:    time.Time{},
		buffer:            buffer,
		minAdjustInterval: minAdjustInterval,
	}, nil
}
