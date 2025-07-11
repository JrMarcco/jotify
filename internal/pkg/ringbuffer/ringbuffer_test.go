package ringbuffer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestTimeDurationRingBuffer_Add(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name       string
		bufferSize int
		items      []time.Duration
		wantSize   int
		wantCount  int
		wantAvg    time.Duration
		wantErr    error
	}{
		{
			name:       "invalid buffer size",
			bufferSize: 0,
			wantErr:    ErrInvalidBufferSize,
		}, {
			name:       "empty buffer",
			bufferSize: 4,
			items:      []time.Duration{},
			wantSize:   4,
			wantCount:  0,
			wantAvg:    0,
			wantErr:    nil,
		}, {
			name:       "with buffer size",
			bufferSize: 4,
			items: []time.Duration{
				time.Second,
				2 * time.Second,
				time.Minute,
			},
			wantSize:  4,
			wantCount: 3,
			wantAvg:   21 * time.Second,
			wantErr:   nil,
		}, {
			name:       "over buffer size",
			bufferSize: 4,
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
			wantErr:   nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buffer, err := NewTimeDurationRingBuffer(tc.bufferSize)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}

			for _, item := range tc.items {
				buffer.Add(item)
			}

			assert.Equal(t, tc.wantSize, buffer.Size())
			assert.Equal(t, tc.wantCount, buffer.Count())
			assert.Equal(t, tc.wantAvg, buffer.Avg())
		})
	}
}

func TestTimeDurationRingBuffer_Reset(t *testing.T) {
	t.Parallel()

	buffer, err := NewTimeDurationRingBuffer(4)
	require.NoError(t, err)

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

	buffer, err := NewTimeDurationRingBuffer(128)
	require.NoError(t, err)

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
