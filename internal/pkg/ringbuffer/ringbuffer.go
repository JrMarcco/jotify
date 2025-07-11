package ringbuffer

import (
	"fmt"
	"sync"
	"time"
)

var (
	ErrInvalidBufferSize = fmt.Errorf("[jotify] buffer size must be greater than zero")
)

// TimeDurationRingBuffer 一个固定大小、线程安全的 time.Duration 环形 buffer 实现。
type TimeDurationRingBuffer struct {
	mu sync.RWMutex

	buffer []time.Duration

	size     int
	count    int
	writePos int

	sum time.Duration
}

func (rb *TimeDurationRingBuffer) Add(d time.Duration) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == rb.size {
		rb.sum -= rb.buffer[rb.writePos]
	} else {
		rb.count++
	}

	rb.buffer[rb.writePos] = d
	rb.sum += d
	rb.writePos = (rb.writePos + 1) % rb.size
}

func (rb *TimeDurationRingBuffer) Avg() time.Duration {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return 0
	}
	return rb.sum / time.Duration(rb.count)
}

func (rb *TimeDurationRingBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for i := 0; i < rb.size; i++ {
		rb.buffer[i] = 0
	}
	rb.sum = 0
	rb.count = 0
	rb.writePos = 0
}

func (rb *TimeDurationRingBuffer) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

func (rb *TimeDurationRingBuffer) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

func NewTimeDurationRingBuffer(size int) (*TimeDurationRingBuffer, error) {
	if size <= 0 {
		return nil, ErrInvalidBufferSize
	}

	return &TimeDurationRingBuffer{
		buffer: make([]time.Duration, size),
		size:   size,
	}, nil
}
