package ioc

import (
	"context"
	"strconv"
	"time"

	"github.com/JrMarcco/dlock"
	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/pkg/batch"
	"github.com/JrMarcco/jotify/internal/pkg/bitring"
	"github.com/JrMarcco/jotify/internal/pkg/job"
	shardingpkg "github.com/JrMarcco/jotify/internal/pkg/sharding"
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/service/channel"
	"github.com/JrMarcco/jotify/internal/service/conf"
	"github.com/JrMarcco/jotify/internal/service/notification"
	"github.com/JrMarcco/jotify/internal/service/notification/callback"
	"github.com/JrMarcco/jotify/internal/service/provider"
	"github.com/JrMarcco/jotify/internal/service/provider/selector"
	"github.com/JrMarcco/jotify/internal/service/provider/sms"
	"github.com/JrMarcco/jotify/internal/service/provider/sms/client"
	"github.com/JrMarcco/jotify/internal/service/schedule"
	shardingsvc "github.com/JrMarcco/jotify/internal/service/schedule/sharding"
	"github.com/JrMarcco/jotify/internal/service/sender"
	"github.com/JrMarcco/jotify/internal/service/sendstrategy"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ServiceFxOpt = fx.Options(
	fx.Provide(
		// biz config service
		fx.Annotate(
			conf.NewDefaultBizConfService,
			fx.As(new(conf.BizConfService)),
		),
		// default send strategy
		fx.Annotate(
			sendstrategy.NewDefaultSendStrategy,
			fx.As(new(sendstrategy.SendStrategy)),
			fx.ResultTags(`name:"default_send_strategy"`),
		),
		// immediate send strategy
		fx.Annotate(
			sendstrategy.NewImmediateSendStrategy,
			fx.As(new(sendstrategy.SendStrategy)),
			fx.ResultTags(`name:"immediate_send_strategy"`),
		),
		fx.Annotate(
			sendstrategy.NewDispatcher,
			fx.As(new(sendstrategy.SendStrategy)),
			fx.ParamTags(`name:"default_send_strategy"`, `name:"immediate_send_strategy"`),
			fx.ResultTags(`name:"send_strategy_dispatcher"`),
		),
		// notification sends service
		fx.Annotate(
			notification.NewDefaultSendService,
			fx.As(new(notification.SendService)),
			fx.ParamTags(`name:"send_strategy_dispatcher"`),
		),
		// callback service
		fx.Annotate(
			callback.NewDefaultService,
			fx.As(new(callback.Service)),
		),
		// provider
		InitTencentSmsClient,
		fx.Annotate(
			InitTencentSmsProvider,
			fx.As(new(provider.Provider)),
			fx.ResultTags(`group:"sms_provider"`),
		),

		// sms channel
		fx.Annotate(
			selector.NewSeqSelectorBuilder,
			fx.As(new(provider.SelectorBuilder)),
			fx.ParamTags(`group:"sms_provider"`),
		),
		channel.NewSmsChannel,
		// channel dispatcher
		InitChannelMap,
		channel.NewDispatcher,

		// notification sender
		fx.Annotate(
			sender.NewDefaultSender,
			fx.As(new(sender.Sender)),
		),

		// notification scheduler
		fx.Annotate(
			InitNotificationScheduler,
			fx.As(new(schedule.NotifScheduler)),
			fx.ParamTags(`name:"notification_sharding_strategy"`),
		),
	),
)

func InitChannelMap(sms *channel.SmsChannel) map[domain.Channel]channel.Channel {
	return map[domain.Channel]channel.Channel{
		domain.ChannelSMS: sms,
	}
}

func InitTencentSmsClient() *client.TencentSmsClient {
	type config struct {
		RegionId  string `mapstructure:"region_id"`
		AppId     string `mapstructure:"app_id"`
		SecretId  string `mapstructure:"secret_id"`
		SecretKey string `mapstructure:"secret_key"`
	}

	cfg := &config{}
	if err := viper.UnmarshalKey("sms.tencent", cfg); err != nil {
		panic(err)
	}

	return client.NewTencentSmsClient(cfg.RegionId, cfg.AppId, cfg.SecretId, cfg.SecretKey)
}

func InitTencentSmsProvider(
	client client.SmsClient, tplRepo repository.ChannelTplRepo, providerRepo repository.ProviderRepo,
) *sms.Provider {
	return sms.NewProvider(
		"tencent_sms_provider",
		client,
		tplRepo,
		providerRepo,
	)
}

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
		BitRingSize          int     `mapstructure:"bit_ring_size"`
		ConsecutiveThreshold int     `mapstructure:"consecutive_threshold"`
		EventRateThreshold   float64 `mapstructure:"event_rate_threshold"`
	}

	type ShardingSchedulerConfig struct {
		MaxLockedTableCntKey string         ``                                    // 最大锁定表数量配置中心 key
		MaxLockedTableCnt    int            `mapstructure:"max_locked_table_cnt"` // 最大锁定表数量
		MinScheduleInterval  time.Duration  `mapstructure:"min_schedule_interval"`
		BatchSize            uint64         `mapstructure:"batch_size"`
		AdjusterConfig       AdjusterConfig `mapstructure:"adjuster_config"`
		ErrEventConfig       ErrEventConfig `mapstructure:"err_event_config"`
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

	adjuster, err := batch.NewSlideWindowAdjuster(
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

	return shardingsvc.NewNotifShardingScheduler(
		dclient,
		notifRepo,
		notifSender,
		shardingStrategy,
		resourceSemaphore,
		cfg.MinScheduleInterval,
		cfg.BatchSize,
		adjuster,
		bitring.NewBitRing(
			cfg.ErrEventConfig.BitRingSize,
			cfg.ErrEventConfig.ConsecutiveThreshold,
			cfg.ErrEventConfig.EventRateThreshold,
		),
		logger,
	)
}
