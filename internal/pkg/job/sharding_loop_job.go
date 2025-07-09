package job

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JrMarcco/dlock"
	"github.com/JrMarcco/jotify/internal/pkg/sharding"
	"go.uber.org/zap"
)

type bizFunc func(ctx context.Context) error

type ShardingLoopJob struct {
	baseKey        string
	retryInterval  time.Duration
	defaultTimeout time.Duration

	resourceSemaphore ResourceSemaphore

	shardingStrategy sharding.Strategy
	dclient          dlock.Dclient
	logger           *zap.Logger

	bizFunc bizFunc
}

func (lj *ShardingLoopJob) Run(ctx context.Context) error {
	for {
		for _, dst := range lj.shardingStrategy.BroadCast() {
			// 抢占任务
			err := lj.resourceSemaphore.Acquire(ctx)
			if err != nil {
				// 超过抢占上限
				time.Sleep(lj.retryInterval)
				continue
			}

			// 尝试获加锁
			dlKey := lj.generateDlockKey(dst.DB, dst.Table)
			// 创建锁
			dl, err := lj.dclient.NewDlock(ctx, dlKey, lj.retryInterval)
			if err != nil {
				// failed to acquire distributed lock
				lj.logger.Error("[jotify] failed to create distributed lock", zap.Error(err))
				err = lj.resourceSemaphore.Release(ctx)
				if err != nil {
					// 释放表的信号量失败
					lj.logger.Error("[jotify] failed to release table semaphore", zap.Error(err))
				}
				continue
			}

			lockCtx, cancel := context.WithTimeout(ctx, lj.defaultTimeout)
			// 尝试加锁（实际获取分布式锁）
			err = dl.TryLock(lockCtx)
			cancel()

			if err != nil {
				// 没抢到分布式锁
				lj.logger.Error("[jotify] failed to acquire distributed lock", zap.Error(err))
				err = lj.resourceSemaphore.Release(ctx)
				if err != nil {
					lj.logger.Error("[jotify] failed to release table semaphore", zap.Error(err))
				}
				continue
			}

			// 成功获取分布式锁
			go lj.tableLoop(sharding.ContextWitDst(ctx, dst), dl)
		}
	}
}

func (lj *ShardingLoopJob) tableLoop(ctx context.Context, dl dlock.Dlock) {
	defer func() {
		if err := lj.resourceSemaphore.Release(ctx); err != nil {
			lj.logger.Error("[jotify] failed to release table semaphore", zap.Error(err))
		}
	}()

	err := lj.bizLoop(ctx, dl)
	// 任务失败可能是 ctx 超时过期，或者是分布式锁续约失败
	if err != nil {
		lj.logger.Error("[jotify] biz loop failed", zap.Error(err))
	}

	// 释放分布式锁
	unlockCtx, cancel := context.WithTimeout(ctx, lj.defaultTimeout)
	unLockErr := dl.Unlock(unlockCtx)
	cancel()

	if unLockErr != nil {
		lj.logger.Error("[jotify] failed to release distributed lock", zap.Error(unLockErr))
	}

	err = ctx.Err()
	if err == nil {
		return
	}
	
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		lj.logger.Info("[jotify] biz loop canceled", zap.Error(err))
		return
	default:
		// 无可挽回的错误
		lj.logger.Error("[jotify] biz loop failed, wait for retry", zap.Error(err))
		time.Sleep(lj.retryInterval)

	}
}

func (lj *ShardingLoopJob) bizLoop(ctx context.Context, dl dlock.Dlock) error {
	const bizTimeout = time.Minute
	for {
		bizCtx, cancel := context.WithTimeout(ctx, bizTimeout)
		err := lj.bizFunc(bizCtx)
		cancel()

		if err != nil {
			lj.logger.Error("[jotify] biz func failed", zap.Error(err))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 分布式锁续约
		refreshCtx, cancel := context.WithTimeout(ctx, lj.retryInterval)
		err = dl.Refresh(refreshCtx)
		cancel()

		if err != nil {
			return fmt.Errorf("[jotify] failed to refresh distributed lock: %w", err)
		}
	}
}

func (lj *ShardingLoopJob) generateDlockKey(db, table string) string {
	return fmt.Sprintf("%s:%s:%s", lj.baseKey, db, table)
}

func NewShardingLoopJob(
	baseKey string,
	resourceSemaphore ResourceSemaphore,
	shardingStrategy sharding.Strategy,
	dclient dlock.Dclient,
	logger *zap.Logger,
	bizFunc bizFunc) *ShardingLoopJob {
	const defaultTimeout = 3 * time.Second
	return newShardingLoopJob(
		baseKey, time.Minute, defaultTimeout, resourceSemaphore, shardingStrategy, dclient, logger, bizFunc,
	)
}

func newShardingLoopJob(
	baseKey string,
	retryInterval time.Duration,
	defaultTimeout time.Duration,
	resourceSemaphore ResourceSemaphore,
	shardingStrategy sharding.Strategy,
	dclient dlock.Dclient,
	logger *zap.Logger,
	bizFunc func(ctx context.Context) error,
) *ShardingLoopJob {
	return &ShardingLoopJob{
		baseKey:           baseKey,
		retryInterval:     retryInterval,
		defaultTimeout:    defaultTimeout,
		resourceSemaphore: resourceSemaphore,
		shardingStrategy:  shardingStrategy,
		dclient:           dclient,
		logger:            logger,
		bizFunc:           bizFunc,
	}
}
