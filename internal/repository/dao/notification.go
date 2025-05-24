package dao

import (
	"github.com/JrMarcco/easy-kit/xsync"
	"github.com/JrMarcco/jotify/internal/pkg/sharding"
	"github.com/JrMarcco/jotify/internal/pkg/snowflake"
	"gorm.io/gorm"
)

// Notification 消息实体
type Notification struct {
	Id            uint64
	BizId         uint64
	BizKey        string
	Receivers     string
	Channel       string
	TplId         uint64
	TplVersionId  uint64
	TplParams     string
	Status        string
	ScheduleStrat int64
	ScheduleEnd   int64
	Version       int32
	CreatedAt     int64
	UpdatedAt     int64
}

type NotificationDAO interface {
}

// NotifShardingDAO NotificationDAO 的分库分表实现
type NotifShardingDAO struct {
	dbs *xsync.Map[string, *gorm.DB]

	notifShardingStrategy sharding.Strategy

	idGenerator *snowflake.Generator
}

func NewNotifShardingDAO(
	dbs *xsync.Map[string, *gorm.DB],
	notifShardingStrategy sharding.Strategy,
	idGenerator *snowflake.Generator,
) *NotifShardingDAO {
	return &NotifShardingDAO{
		dbs:                   dbs,
		notifShardingStrategy: notifShardingStrategy,
		idGenerator:           idGenerator,
	}
}
