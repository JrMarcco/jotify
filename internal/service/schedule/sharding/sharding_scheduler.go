package sharding

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/JrMarcco/dlock"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/pkg/batch"
	"github.com/JrMarcco/jotify/internal/pkg/bitring"
	"github.com/JrMarcco/jotify/internal/pkg/job"
	"github.com/JrMarcco/jotify/internal/pkg/sharding"
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/service/schedule"
	"github.com/JrMarcco/jotify/internal/service/sender"
	"go.uber.org/zap"
)

var _ schedule.NotifScheduler = (*NotifShardingScheduler)(nil)

// NotifShardingScheduler 通知分库分表调度器
type NotifShardingScheduler struct {
	notifRepo   repository.NotificationRepo
	notifSender sender.Sender

	loopInterval time.Duration

	batchSize     atomic.Uint64
	batchAdjuster batch.Adjuster

	errEvents *bitring.BitRing
	job       *job.ShardingLoopJob
}

// Start 启动调度服务
// context.Context 被取消或关闭时退出调度循环。
func (ss *NotifShardingScheduler) Start(ctx context.Context) error {
	go func() {
		_ = ss.job.Run(ctx)
	}()
	return nil
}

func (ss *NotifShardingScheduler) loop(ctx context.Context) error {
	for {
		start := time.Now()

		cnt, sendErr := ss.batchSend(ctx)

		// 记录响应时间
		respTime := time.Since(start)

		// 记录错误事件
		ss.errEvents.Add(sendErr != nil)
		// 错误事件触发阈值（连续错误事件或错误率）
		if ss.errEvents.ThresholdTriggering() {
			return errs.ErrEventThresholdExceeded
		}

		newBatchSize, adjustErr := ss.batchAdjuster.Adjust(ctx, respTime)
		if adjustErr == nil {
			ss.batchSize.Store(newBatchSize)
		}

		// 没有数据时，等待一段时间再进行下一次调度
		if cnt == 0 {
			time.Sleep(ss.loopInterval - respTime)
			continue
		}
	}
}

// batchSend 批量发送已就绪的通知
// 执行成功会返回发送的通知数量
func (ss *NotifShardingScheduler) batchSend(ctx context.Context) (int, error) {
	const defaultTimeout = 3 * time.Second

	loopCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	const offset = 0
	notifications, err := ss.notifRepo.FindReady(loopCtx, offset, int(ss.batchSize.Load()))
	if err != nil {
		return 0, err
	}

	if len(notifications) == 0 {
		return 0, nil
	}

	_, err = ss.notifSender.BatchSend(loopCtx, notifications)
	return len(notifications), err
}

func NewNotifShardingScheduler(
	dclient dlock.Dclient,
	notifRepo repository.NotificationRepo,
	notifSender sender.Sender,
	shardingStrategy sharding.Strategy,
	resourceSemaphore job.ResourceSemaphore,
	loopInterval time.Duration,
	batchSize uint64,
	batchAdjuster batch.Adjuster,
	errEvents *bitring.BitRing,
	logger *zap.Logger,
) *NotifShardingScheduler {
	const jobBaseKey = "jotify_async_sharding_scheduler"

	scheduler := &NotifShardingScheduler{
		notifRepo:     notifRepo,
		notifSender:   notifSender,
		loopInterval:  loopInterval,
		batchAdjuster: batchAdjuster,
		errEvents:     errEvents,
	}
	scheduler.job = job.NewShardingLoopJob(
		jobBaseKey, resourceSemaphore, shardingStrategy, dclient, logger, scheduler.loop,
	)
	scheduler.batchSize.Store(batchSize)
	return scheduler
}
