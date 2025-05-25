package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JrMarcco/easy-kit/xsync"
	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/pkg/sharding"
	"github.com/JrMarcco/jotify/internal/pkg/snowflake"
	"github.com/go-sql-driver/mysql"
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
	Create(ctx context.Context, n Notification) (Notification, error)
}

var _ NotificationDAO = (*NotifShardingDAO)(nil)

// NotifShardingDAO NotificationDAO 的分库分表实现
type NotifShardingDAO struct {
	dbs *xsync.Map[string, *gorm.DB]

	notifShardingStrategy sharding.Strategy
	cbLogShardingStrategy sharding.Strategy

	idGenerator *snowflake.Generator
}

func (nd *NotifShardingDAO) Create(ctx context.Context, n Notification) (Notification, error) {
	return nd.create(ctx, n, false)
}

func (nd *NotifShardingDAO) create(ctx context.Context, n Notification, needCallback bool) (Notification, error) {
	now := time.Now().UnixMilli()

	n.CreatedAt, n.UpdatedAt, n.Version = now, now, 1

	// 分库分表规则
	notifDst := nd.notifShardingStrategy.Shard(n.BizId, n.BizKey)
	cbLogDst := nd.cbLogShardingStrategy.Shard(n.BizId, n.BizKey)

	db, ok := nd.dbs.Load(notifDst.DB)
	if !ok {
		return Notification{}, fmt.Errorf("fail to load db: %s", notifDst.DB)
	}

	// 业务上指定 notification 和 callback_log 使用相同的分库规则，即在同一个库中
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for {
			n.Id = nd.idGenerator.NextId(n.BizId, n.BizKey)
			if err := tx.Table(notifDst.Table).Create(&n).Error; err != nil {
				// 创建 notification 记录失败
				if nd.isUniqueConstraintErr(err) {
					// 唯一索引冲突
					if IsIdDuplicateErr(n.Id, err) {
						// 主键冲突，重新生成 id 再执行插入
						continue
					}
					// 非主键冲突直接返回
					return nil
				}
				return err
			}

			if needCallback {
				cb := &CallbackLog{
					Id:          n.Id,
					Status:      domain.CallbackStatusInit.String(),
					NextRetryAt: now,
					CreatedAt:   now,
					UpdatedAt:   now,
				}
				if err := tx.Table(cbLogDst.Table).Create(&cb).Error; err != nil {
					return fmt.Errorf("%w", errs.ErrFailedToCreateCallbackLog)
				}
			}
			return nil
		}
	})
	return n, err
}

func (nd *NotifShardingDAO) isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}

	mysqlErr := new(mysql.MySQLError)
	if ok := errors.As(err, &mysqlErr); ok {
		const uniqueConstraintErrCode = 1062
		return mysqlErr.Number == uniqueConstraintErrCode
	}
	return false
}

func NewNotifShardingDAO(
	dbs *xsync.Map[string, *gorm.DB],
	notifShardingStrategy sharding.Strategy,
	cbLogShardingStrategy sharding.Strategy,
	idGenerator *snowflake.Generator,
) *NotifShardingDAO {
	return &NotifShardingDAO{
		dbs:                   dbs,
		notifShardingStrategy: notifShardingStrategy,
		cbLogShardingStrategy: cbLogShardingStrategy,
		idGenerator:           idGenerator,
	}
}
