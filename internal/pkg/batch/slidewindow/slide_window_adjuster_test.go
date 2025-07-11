package slidewindow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBufferAdjuster_Adjust(t *testing.T) {
	tcs := []struct {
		name     string
		adjuster *Adjuster
		respTime time.Duration
		wantSize int
	}{
		{
			name: "buffer not full",
			adjuster: func() *Adjuster {
				adjuster, err := NewAdjuster(
					4, 100, 50, 200, 10, time.Second,
				)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)

				return adjuster
			}(),
			respTime: time.Millisecond,
			wantSize: 100,
		}, {
			name: "equal avg time",
			adjuster: func() *Adjuster {
				adjuster, err := NewAdjuster(
					4, 100, 50, 200, 10, time.Second,
				)

				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)

				return adjuster
			}(),
			respTime: time.Millisecond,
			wantSize: 100,
		}, {
			name: "faster than avg time",
			adjuster: func() *Adjuster {
				adjuster, err := NewAdjuster(
					4, 100, 50, 200, 10, time.Second,
				)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), 2*time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), 2*time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), 2*time.Millisecond)
				require.NoError(t, err)

				return adjuster
			}(),
			respTime: time.Millisecond,
			wantSize: 110,
		}, {
			name: "faster within adjust interval",
			adjuster: func() *Adjuster {
				adjuster, err := NewAdjuster(
					4, 100, 50, 200, 10, time.Minute,
				)
				_, err = adjuster.Adjust(context.Background(), time.Second)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Second)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Second)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)

				return adjuster
			}(),
			respTime: time.Millisecond,
			wantSize: 110,
		}, {
			name: "slower than avg time",
			adjuster: func() *Adjuster {
				adjuster, err := NewAdjuster(
					4, 100, 50, 200, 10, time.Second,
				)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)

				return adjuster
			}(),
			respTime: time.Second,
			wantSize: 90,
		}, {
			name: "slower within adjust interval",
			adjuster: func() *Adjuster {
				adjuster, err := NewAdjuster(
					4, 100, 50, 200, 10, time.Minute,
				)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), 10*time.Millisecond)
				require.NoError(t, err)

				return adjuster
			}(),
			respTime: time.Second,
			wantSize: 90,
		}, {
			name: "more than max batch size",
			adjuster: func() *Adjuster {
				adjuster, err := NewAdjuster(
					4, 100, 50, 200, 80, time.Millisecond,
				)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Second)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Second)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Second)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)

				return adjuster
			}(),
			respTime: time.Millisecond,
			wantSize: 200,
		}, {
			name: "less than min batch size",
			adjuster: func() *Adjuster {
				adjuster, err := NewAdjuster(
					4, 100, 50, 200, 40, time.Millisecond,
				)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), time.Millisecond)
				require.NoError(t, err)
				_, err = adjuster.Adjust(context.Background(), 10*time.Millisecond)
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
