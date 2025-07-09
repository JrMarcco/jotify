package buffer

import (
	"sync"
	"time"
)

// TimeDurationRingBuffer 一个固定大、线程安全的 time.Duration 环形缓冲。
type TimeDurationRingBuffer struct {
	mu sync.RWMutex

	size     int
	count    int
	writePos int

	buffer []time.Duration
	sum    time.Duration
}

func (b *TimeDurationRingBuffer) Add(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == b.size {
		// buffer 满了，移除最早的元素（环结构）
		b.sum -= b.buffer[b.writePos]
	} else {
		b.count++
	}

	b.buffer[b.writePos] = d
	b.sum += d

	// 移动指针
	b.writePos = (b.writePos + 1) % b.size
}

func (b *TimeDurationRingBuffer) Avg() time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.count == 0 {
		return 0
	}
	return b.sum / time.Duration(b.count)
}

func (b *TimeDurationRingBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := 0; i < b.size; i++ {
		b.buffer[i] = 0
	}

	b.sum = 0
	b.count = 0
	b.writePos = 0
}

func (b *TimeDurationRingBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size
}

func (b *TimeDurationRingBuffer) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.count
}

func NewTimeDurationRingBuffer(size int) *TimeDurationRingBuffer {
	if size <= 0 {
		panic("[jotify] buffer size must be greater than zero")
	}

	return &TimeDurationRingBuffer{
		buffer: make([]time.Duration, size),
		size:   size,
	}
}
