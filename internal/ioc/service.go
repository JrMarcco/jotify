package ioc

import (
	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/service/channel"
	"github.com/JrMarcco/jotify/internal/service/conf"
	"github.com/JrMarcco/jotify/internal/service/notification"
	"github.com/JrMarcco/jotify/internal/service/provider"
	"github.com/JrMarcco/jotify/internal/service/provider/selector"
	"github.com/JrMarcco/jotify/internal/service/provider/sms"
	"github.com/JrMarcco/jotify/internal/service/provider/sms/client"
	"github.com/JrMarcco/jotify/internal/service/sender"
	"github.com/JrMarcco/jotify/internal/service/sendstrategy"
	"github.com/spf13/viper"
	"go.uber.org/fx"
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
			notification.NewDefaultCallbackService,
			fx.As(new(notification.CallbackService)),
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
