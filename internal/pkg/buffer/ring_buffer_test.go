package buffer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func TestNewTimeDurationRingBuffer(t *testing.T) {
	assert.Panics(t, func() {
		NewTimeDurationRingBuffer(0)
	})

	assert.Panics(t, func() {
		NewTimeDurationRingBuffer(-1)
	})

	assert.NotPanics(t, func() {
		NewTimeDurationRingBuffer(1)
	})
}

func TestTimeDurationRingBuffer_Add(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name      string
		buffer    *TimeDurationRingBuffer
		items     []time.Duration
		wantSize  int
		wantCount int
		wantAvg   time.Duration
	}{
		{
			name:      "empty buffer",
			buffer:    NewTimeDurationRingBuffer(4),
			items:     []time.Duration{},
			wantSize:  4,
			wantCount: 0,
			wantAvg:   0,
		}, {
			name:   "with buffer size",
			buffer: NewTimeDurationRingBuffer(4),
			items: []time.Duration{
				time.Second,
				2 * time.Second,
				time.Minute,
			},
			wantSize:  4,
			wantCount: 3,
			wantAvg:   21 * time.Second,
		}, {
			name:   "over buffer size",
			buffer: NewTimeDurationRingBuffer(4),
			items: []time.Duration{
				time.Second,
				time.Second,
				time.Second,
				time.Second,
				time.Second,
				time.Minute,
				time.Minute,
			},
			wantSize:  4,
			wantCount: 4,
			wantAvg:   30500 * time.Millisecond,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, item := range tc.items {
				tc.buffer.Add(item)
			}

			assert.Equal(t, tc.wantSize, tc.buffer.Size())
			assert.Equal(t, tc.wantCount, tc.buffer.Count())
			assert.Equal(t, tc.wantAvg, tc.buffer.Avg())
		})
	}
}

func TestTimeDurationRingBuffer_Reset(t *testing.T) {
	t.Parallel()

	buffer := NewTimeDurationRingBuffer(4)

	buffer.Add(time.Second)
	buffer.Add(2 * time.Second)
	buffer.Add(time.Minute)
	assert.Equal(t, 3, buffer.Count())
	assert.Equal(t, 4, buffer.Size())
	assert.Equal(t, 21*time.Second, buffer.Avg())

	buffer.Reset()
	assert.Equal(t, 0, buffer.Count())
	assert.Equal(t, 4, buffer.Size())
	assert.Equal(t, time.Duration(0), buffer.Avg())

	buffer.Add(time.Second)
	buffer.Add(3 * time.Second)
	assert.Equal(t, 2, buffer.Count())
	assert.Equal(t, 4, buffer.Size())
	assert.Equal(t, 2*time.Second, buffer.Avg())
}

func TestTimeDurationRingBuffer_ThreadSafe(t *testing.T) {
	t.Parallel()

	buffer := NewTimeDurationRingBuffer(128)

	var eg errgroup.Group
	for i := 0; i < 128; i++ {
		num := i
		eg.Go(func() error {
			buffer.Add(time.Duration(num) * time.Millisecond)
			_ = buffer.Avg()
			return nil
		})
	}
	_ = eg.Wait()

	assert.Equal(t, 128, buffer.Size())
	assert.Equal(t, 128, buffer.Count())
}
