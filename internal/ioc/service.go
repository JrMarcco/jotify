package ioc

import (
	"github.com/JrMarcco/jotify/internal/service/conf"
	"github.com/JrMarcco/jotify/internal/service/notification"
	"github.com/JrMarcco/jotify/internal/service/sendstrategy"
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
	),
)
