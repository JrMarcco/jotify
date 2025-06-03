package ioc

import (
	"github.com/JrMarcco/jotify/internal/repository"
	"github.com/JrMarcco/jotify/internal/repository/cache"
	"github.com/JrMarcco/jotify/internal/repository/cache/local"
	"github.com/JrMarcco/jotify/internal/repository/cache/redis"
	"github.com/JrMarcco/jotify/internal/repository/dao"
	"go.uber.org/fx"
)

var RepoFxOpt = fx.Options(
	// cache
	fx.Provide(
		fx.Annotate(
			local.NewBizConfLocalCache,
			fx.As(new(cache.BizConfCache)),
			fx.ResultTags(`name:"biz_conf_local_cache"`),
		),
		fx.Annotate(
			redis.NewBizConfRedisCache,
			fx.As(new(cache.BizConfCache)),
			fx.ResultTags(`name:"biz_conf_redis_cache"`),
		),
		fx.Annotate(
			redis.NewQuotaRedisCache,
			fx.As(new(cache.QuotaCache)),
			fx.ResultTags(`name:"quota_redis_cache"`),
		),
	),

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
			fx.As(new(dao.NotificationDAO)),
			fx.ParamTags(`name:"notification_sharding_strategy"`, `name:"callback_log_sharding_strategy"`),
		),
		// channel template dao
		fx.Annotate(
			dao.NewDefaultChannelTplDAO,
			fx.As(new(dao.ChannelTplDAO)),
		),
		// provider dao
		fx.Annotate(
			dao.NewDefaultProviderDAO,
			fx.As(new(dao.ProviderDAO)),
		),
		// callback log dao
		fx.Annotate(
			dao.NewDefaultCallbackLogDAO,
			fx.As(new(dao.CallbackLogDAO)),
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
		// channel template repository
		fx.Annotate(
			repository.NewDefaultChannelTplRepo,
			fx.As(new(repository.ChannelTplRepo)),
		),
		// provider repository
		fx.Annotate(
			repository.NewDefaultProviderRepo,
			fx.As(new(repository.ProviderRepo)),
		),
		// callback log repository
		fx.Annotate(
			repository.NewDefaultCallbackLogRepo,
			fx.As(new(repository.CallbackLogRepo)),
		),
	),
)
