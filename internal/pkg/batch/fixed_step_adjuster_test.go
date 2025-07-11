package batch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedStepAdjuster_Adjust(t *testing.T) {
	tcs := []struct {
		name     string
		adjuster *FixedStepAdjuster
		respTime time.Duration
		wantSize int
	}{
		{
			name: "without adjust",
			adjuster: NewFixedStepAdjuster(
				100, 50, 200, 10,
				time.Second, time.Millisecond, 10*time.Millisecond,
			),
			respTime: 2 * time.Millisecond,
			wantSize: 100,
		}, {
			name: "equal fast resp time",
			adjuster: NewFixedStepAdjuster(
				100, 50, 200, 10,
				time.Second, time.Millisecond, 10*time.Millisecond,
			),
			respTime: time.Millisecond,
			wantSize: 100,
		}, {
			name: "equal slow resp time",
			adjuster: NewFixedStepAdjuster(
				100, 50, 200, 10,
				time.Second, time.Millisecond, 10*time.Millisecond,
			),
			respTime: 10 * time.Millisecond,
			wantSize: 100,
		}, {
			name: "faster than fast resp time",
			adjuster: NewFixedStepAdjuster(
				100, 50, 200, 10,
				time.Second, time.Millisecond, 10*time.Millisecond,
			),
			respTime: time.Nanosecond,
			wantSize: 110,
		}, {
			name: "slower than slow resp time",
			adjuster: NewFixedStepAdjuster(
				100, 50, 200, 10,
				time.Second, time.Millisecond, 10*time.Millisecond,
			),
			respTime: time.Second,
			wantSize: 90,
		}, {
			name: "faster within adjust interval",
			adjuster: func() *FixedStepAdjuster {
				adjuster := NewFixedStepAdjuster(
					100, 50, 200, 10,
					time.Minute, time.Millisecond, 10*time.Millisecond,
				)
				_, err := adjuster.Adjust(context.Background(), time.Nanosecond)
				require.NoError(t, err)
				return adjuster
			}(),
			respTime: time.Nanosecond,
			wantSize: 110,
		}, {
			name: "slower within adjust interval",
			adjuster: func() *FixedStepAdjuster {
				adjuster := NewFixedStepAdjuster(
					100, 50, 200, 10,
					time.Minute, time.Millisecond, 10*time.Millisecond,
				)
				_, err := adjuster.Adjust(context.Background(), 11*time.Millisecond)
				require.NoError(t, err)
				return adjuster
			}(),
			respTime: time.Second,
			wantSize: 90,
		}, {
			name: "more than max batch size",
			adjuster: func() *FixedStepAdjuster {
				adjuster := NewFixedStepAdjuster(
					100, 50, 200, 80,
					time.Millisecond, time.Second, 10*time.Second,
				)
				_, err := adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				return adjuster
			}(),
			respTime: time.Millisecond,
			wantSize: 200,
		}, {
			name: "less than min batch size",
			adjuster: func() *FixedStepAdjuster {
				adjuster := NewFixedStepAdjuster(
					100, 50, 200, 30,
					time.Millisecond, time.Millisecond, 10*time.Millisecond,
				)
				_, err := adjuster.Adjust(context.Background(), time.Second)
				require.NoError(t, err)
				return adjuster
			}(),
			respTime: time.Second,
			wantSize: 50,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			time.Sleep(time.Millisecond)
			size, err := tc.adjuster.Adjust(context.Background(), tc.respTime)
			require.NoError(t, err)
			assert.Equal(t, tc.wantSize, size)
		})
	}
}
