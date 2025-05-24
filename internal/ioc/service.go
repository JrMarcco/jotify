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
		),
		// notification sends service
		fx.Annotate(
			notification.NewDefaultSendService,
			fx.As(new(notification.SendService)),
		),
	),
)
