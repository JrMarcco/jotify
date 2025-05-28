package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/JrMarcco/easy-kit/slice"
	"github.com/JrMarcco/easy-kit/xmap"
	"github.com/JrMarcco/easy-kit/xsync"
	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/pkg/sharding"
	"github.com/JrMarcco/jotify/internal/pkg/snowflake"
	"golang.org/x/sync/errgroup"
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
	CreateWithCallback(ctx context.Context, entity Notification) (Notification, error)
	BatchCreate(ctx context.Context, ns []Notification) ([]Notification, error)
	BatchCreateWithCallback(ctx context.Context, ns []Notification) ([]Notification, error)
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

func (nd *NotifShardingDAO) CreateWithCallback(ctx context.Context, entity Notification) (Notification, error) {
	return nd.create(ctx, entity, true)
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
				if errors.Is(err, gorm.ErrDuplicatedKey) && IsIdDuplicateErr([]uint64{n.Id}, err) {
					// 主键冲突，重新生成 id 再执行插入
					continue
				}
				return err
			}

			if needCallback {
				cb := &CallbackLog{
					NotificationId: n.Id,
					Status:         domain.CallbackStatusInit.String(),
					NextRetryAt:    now,
					CreatedAt:      now,
					UpdatedAt:      now,
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

func (nd *NotifShardingDAO) BatchCreate(ctx context.Context, ns []Notification) ([]Notification, error) {
	return nd.batchCreate(ctx, ns, false)
}

func (nd *NotifShardingDAO) BatchCreateWithCallback(ctx context.Context, ns []Notification) ([]Notification, error) {
	return nd.batchCreate(ctx, ns, true)
}

func (nd *NotifShardingDAO) batchCreate(ctx context.Context, ns []Notification, needCallback bool) ([]Notification, error) {
	if len(ns) == 0 {
		return []Notification{}, nil
	}

	now := time.Now().UnixMilli()
	pointers := slice.Map(ns, func(_ int, src Notification) *Notification {
		src.CreatedAt, src.UpdatedAt, src.Version = now, now, 1
		return &src
	})

	return nd.tryBatchInsert(ctx, pointers, needCallback)
}

func (nd *NotifShardingDAO) tryBatchInsert(ctx context.Context, ns []*Notification, needCallback bool) ([]Notification, error) {
	// 按照分库规则对 notification 进行分组
	const maxDBNum = 32
	m := make(map[string][]*Notification, maxDBNum)

	for _, n := range ns {
		dst := nd.notifShardingStrategy.Shard(n.BizId, n.BizKey)
		mapNs, ok := m[dst.DB]
		if !ok {
			mapNs = make([]*Notification, 0, 16)
		}
		mapNs = append(mapNs, n)
		m[dst.DB] = mapNs
	}

	var eg errgroup.Group
	dbNames := xmap.Keys(m)
	for _, dbName := range dbNames {
		// 这里不会出现不存在的情况，可以忽略第二个参数
		dbNs, _ := m[dbName]
		db, ok := nd.dbs.Load(dbName)
		if !ok {
			return []Notification{}, fmt.Errorf("fail to load db: %s", dbName)
		}

		eg.Go(func() error {
			for {
				sql, args, ids := nd.sqlGenerate(db, dbNs, needCallback)
				if sql != "" {
					if err := db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
						if errors.Is(err, gorm.ErrDuplicatedKey) && IsIdDuplicateErr(ids, err) {
							// 主键冲突，重新生成 id 再执行插入
							continue
						}
						return err
					}
				}
				return nil
			}
		})
	}

	err := eg.Wait()
	return slice.Map(ns, func(_ int, src *Notification) Notification {
		if src == nil {
			return Notification{}
		}
		return *src
	}), err
}

func (nd *NotifShardingDAO) sqlGenerate(db *gorm.DB, ns []*Notification, needCallback bool) (string, []any, []uint64) {
	now := time.Now().UnixMilli()

	// 临时开启 gorm dry run 来生成 sql
	gormSession := db.Session(&gorm.Session{DryRun: true})
	ids := make([]uint64, 0, len(ns))
	// 包含 callback log
	sqls := make([]string, 0, 2*len(ns))
	// Notification 14 个字段
	// CallbackLog  6  个字段
	args := make([]any, 0, 20*len(ns))

	for _, n := range ns {
		id := nd.idGenerator.NextId(n.BizId, n.BizKey)
		n.Id = id
		ids = append(ids, id)

		dst := nd.notifShardingStrategy.Shard(n.BizId, n.BizKey)
		statement := gormSession.Table(dst.Table).Create(&n).Statement
		sqls = append(sqls, statement.SQL.String())
		args = append(args, statement.Vars...)

		if needCallback {
			dst = nd.cbLogShardingStrategy.Shard(n.BizId, n.BizKey)
			statement = gormSession.Table(dst.Table).Create(&CallbackLog{
				NotificationId: id,
				Status:         domain.CallbackStatusInit.String(),
				NextRetryAt:    now,
				CreatedAt:      now,
				UpdatedAt:      now,
			}).Statement
			sqls = append(sqls, statement.SQL.String())
			args = append(args, statement.Vars...)
		}
	}

	if len(sqls) == 0 {
		return "", []any{}, []uint64{}
	}

	return strings.Join(sqls, ";"), args, ids
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
