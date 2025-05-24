package ioc

import (
	"github.com/JrMarcco/jotify/internal/repository/dao"
	"go.uber.org/fx"
)

var DaoFxOpt = fx.Provide(
	fx.Annotate(
		dao.NewNotifShardingDAO,
		fx.As(new(dao.NotifShardingDAO)),
	),
)
