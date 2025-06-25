package job

import (
	"context"
	"time"

	"github.com/JrMarcco/dlock"
	"github.com/JrMarcco/jotify/internal/pkg/sharding"
	"go.uber.org/zap"
)

type ShardingLoopJob struct {
	baseKey        string
	retryInterval  time.Duration
	defaultTimeout time.Duration

	semaphore ResourceSemaphore

	strategy sharding.Strategy
	dclient  dlock.Dclient
	logger   *zap.Logger

	bizFunc func(ctx context.Context) error
}

func (lj *ShardingLoopJob) Run() {

}

func NewShardingLoopJob(
	baseKey string,
	bizFunc func(ctx context.Context) error) *ShardingLoopJob {
	return &ShardingLoopJob{
		baseKey: baseKey,
		bizFunc: bizFunc,
	}
}
