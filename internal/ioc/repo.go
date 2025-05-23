package ioc

import (
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/repository/dao"
	"go.uber.org/fx"
)

var RepoFxOpt = fx.Options(
	// dao
	fx.Provide(
		// biz config dao
		fx.Annotate(
			dao.NewDefaultBizConfDAO,
			fx.As(new(dao.BizConfDAO)),
		),
		// notification sharding dao
		fx.Annotate(
			dao.NewNotifShardingDAO,
			fx.As(new(dao.NotifShardingDAO)),
		),
	),

	// repository
	fx.Provide(
		// biz config repository
		fx.Annotate(
			repository.NewDefaultBizConfRepo,
			fx.As(new(repository.BizConfRepo)),
		),
		// notification repository
		fx.Annotate(
			repository.NewDefaultNotifRepo,
			fx.As(new(repository.NotificationRepo)),
		),
	),
)
