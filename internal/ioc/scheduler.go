package ioc

import (
	"context"
	"strconv"
	"time"

	"github.com/JrMarcco/dlock"
	"github.com/JrMarcco/jotify/internal/pkg/batch/slidewindow"
	"github.com/JrMarcco/jotify/internal/pkg/bitring"
	"github.com/JrMarcco/jotify/internal/pkg/job"
	shardingpkg "github.com/JrMarcco/jotify/internal/pkg/sharding"
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/service/schedule"
	shardingsvc "github.com/JrMarcco/jotify/internal/service/schedule/sharding"
	"github.com/JrMarcco/jotify/internal/service/sender"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var SchedulerFxOpt = fx.Options(
	fx.Provide(
		// notification scheduler
		fx.Annotate(
			InitNotificationScheduler,
			fx.As(new(schedule.NotifScheduler)),
			fx.ParamTags(`name:"notification_sharding_strategy"`),
		),
	),
)

func InitNotificationScheduler(
	dclient dlock.Dclient,
	notifRepo repository.NotificationRepo,
	notifSender sender.Sender,
	shardingStrategy shardingpkg.Strategy,
	etcdClient *clientv3.Client,
	logger *zap.Logger,
) schedule.NotifScheduler {
	type AdjusterConfig struct {
		BuffSize          int           `mapstructure:"buff_size"`
		InitSize          uint64        `mapstructure:"init_size"`
		MinSize           uint64        `mapstructure:"min_size"`
		MaxSize           uint64        `mapstructure:"max_size"`
		AdjustStep        uint64        `mapstructure:"adjust_step"`
		MinAdjustInterval time.Duration `mapstructure:"min_adjust_interval"`
	}

	type ErrEventConfig struct {
		BitRingSize          int     `mapstructure:"bit_ring_size"`         // 位环大小
		ConsecutiveThreshold int     `mapstructure:"consecutive_threshold"` // 连续阈值
		EventRateThreshold   float64 `mapstructure:"event_rate_threshold"`  // 事件率阈值
	}

	type ShardingSchedulerConfig struct {
		MaxLockedTableCntKey string         `mapstructure:"max_locked_table_cnt_key"` // 最大锁定表数量配置中心 key
		MaxLockedTableCnt    int            `mapstructure:"max_locked_table_cnt"`     // 最大锁定表数量
		MinScheduleInterval  time.Duration  `mapstructure:"min_schedule_interval"`    // 最小调度间隔
		BatchSize            uint64         `mapstructure:"batch_size"`               // 批量大小
		AdjusterConfig       AdjusterConfig `mapstructure:"adjuster_config"`          // 调整器配置
		ErrEventConfig       ErrEventConfig `mapstructure:"err_event_config"`         // 错误事件配置
	}

	var cfg ShardingSchedulerConfig
	if err := viper.UnmarshalKey("sharding_scheduler", &cfg); err != nil {
		panic(err)
	}

	resourceSemaphore := job.NewMaxCntResourceSemaphore(cfg.MaxLockedTableCnt)
	// 处理最大锁定表数量表更时间
	go func() {
		watchChan := etcdClient.Watch(context.Background(), cfg.MaxLockedTableCntKey)
		for watchResp := range watchChan {
			for _, ev := range watchResp.Events {
				if ev.Type == clientv3.EventTypePut {
					maxLockedTableCnt, _ := strconv.ParseInt(string(ev.Kv.Value), 10, 64)
					resourceSemaphore.UpdateMaxCnt(int(maxLockedTableCnt))
				}
			}
		}
	}()

	adjuster, err := slidewindow.NewAdjuster(
		cfg.AdjusterConfig.BuffSize,
		cfg.AdjusterConfig.InitSize,
		cfg.AdjusterConfig.MinSize,
		cfg.AdjusterConfig.MaxSize,
		cfg.AdjusterConfig.AdjustStep,
		cfg.AdjusterConfig.MinAdjustInterval,
	)
	if err != nil {
		panic(err)
	}

	errEvents := bitring.NewBitRing(
		cfg.ErrEventConfig.BitRingSize,
		cfg.ErrEventConfig.ConsecutiveThreshold,
		cfg.ErrEventConfig.EventRateThreshold,
	)

	return shardingsvc.NewNotifShardingScheduler(
		dclient,
		notifRepo,
		notifSender,
		shardingStrategy,
		resourceSemaphore,
		cfg.MinScheduleInterval,
		cfg.BatchSize,
		adjuster,
		errEvents,
		logger,
	)
}
