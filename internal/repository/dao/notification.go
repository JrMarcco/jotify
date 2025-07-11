package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	BatchUpdateStatus(ctx context.Context, successNs, failureNs []Notification) error

	GetById(ctx context.Context, id uint64) (Notification, error)
	GetByKey(ctx context.Context, bizId uint64, bizKey string) (Notification, error)
	GetMapByIds(ctx context.Context, ids []uint64) (map[uint64]Notification, error)

	MarkSuccess(ctx context.Context, n Notification) error
	MarkFailure(ctx context.Context, n Notification) error

	CompareAndSwapStatus(ctx context.Context, n Notification) error

	FindReady(ctx context.Context, offset int, limit int) ([]Notification, error)
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
		return Notification{}, fmt.Errorf("failed to load db: %s", notifDst.DB)
	}

	// 业务上指定 notification 和 callback_log 使用相同的分库规则，即在同一个库中
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for {
			n.Id = nd.idGenerator.NextId(n.BizId, n.BizKey)
			if err := tx.Table(notifDst.Table).Create(&n).Error; err != nil {
				// 创建 notification 记录失败，如果是主键冲突直接返回错误
				if errors.Is(err, gorm.ErrDuplicatedKey) && IsIdDuplicateErr([]uint64{n.Id}, err) {
					return fmt.Errorf("%w", errs.ErrDuplicateNotificationId)
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
			return []Notification{}, fmt.Errorf("failed to load db: %s", dbName)
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

func (nd *NotifShardingDAO) BatchUpdateStatus(ctx context.Context, successNs, failureNs []Notification) error {
	if len(successNs) == 0 && len(failureNs) == 0 {
		return nil
	}

	dbMap := make(map[string]map[string]*modifyIds)
	for i := range successNs {
		n := successNs[i]
		notifDst := nd.notifShardingStrategy.ShardWithId(n.Id)
		callbackDst := nd.cbLogShardingStrategy.ShardWithId(n.Id)

		tableMap, ok := dbMap[notifDst.DB]
		if !ok {
			tableMap = make(map[string]*modifyIds)
			dbMap[notifDst.DB] = tableMap
		}

		modifyId, ok := tableMap[notifDst.Table]
		if ok {
			modifyId.successIds = append(modifyId.successIds, n.Id)
		} else {
			modifyId = &modifyIds{
				callbackTable: callbackDst.Table,
				successIds:    []uint64{n.Id},
				failureIds:    []uint64{},
			}
			tableMap[notifDst.Table] = modifyId
		}
	}

	for i := range failureNs {
		n := failureNs[i]
		notifDst := nd.notifShardingStrategy.ShardWithId(n.Id)
		callbackDst := nd.cbLogShardingStrategy.ShardWithId(n.Id)

		tableMap, ok := dbMap[notifDst.DB]
		if !ok {
			tableMap = make(map[string]*modifyIds)
			dbMap[notifDst.DB] = tableMap
		}

		modifyId, ok := tableMap[notifDst.Table]
		if ok {
			modifyId.successIds = append(modifyId.successIds, n.Id)
		} else {
			modifyId = &modifyIds{
				callbackTable: callbackDst.Table,
				successIds:    []uint64{},
				failureIds:    []uint64{n.Id},
			}
			tableMap[notifDst.Table] = modifyId
		}
	}

	var eg errgroup.Group
	for dbName, tableMap := range dbMap {
		db := dbName
		tbMap := tableMap
		eg.Go(func() error {
			gormDB, ok := nd.dbs.Load(db)
			if !ok {
				return fmt.Errorf("failed to load db: %s", db)
			}
			return gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				return nd.batchMark(tx, tbMap)
			})
		})
	}
	return eg.Wait()
}

//goland:noinspection SqlNoDataSourceInspection
func (nd *NotifShardingDAO) batchMark(tx *gorm.DB, tbMap map[string]*modifyIds) error {
	now := time.Now().UnixMilli()
	sqls := make([]string, 0, len(tbMap))

	for tb := range tbMap {
		m := tbMap[tb]
		if len(m.successIds) > 0 {
			notifSQL := fmt.Sprintf(
				"UPDATE %s SET `version` = `version` + 1, `status` = '%s', `update_at` = %d WHERE `id` IN (%s)",
				tb, domain.SendStatusSuccess.String(), now, m.successToString(),
			)
			cbLogSQL := fmt.Sprintf(
				"UPDATE %s SET `status` = '%s', `update_at` = %d WHERE `notification_id` IN (%s)",
				tb, domain.CallbackStatusPending.String(), now, m.successToString(),
			)
			sqls = append(sqls, notifSQL, cbLogSQL)
		}
		if len(m.failureIds) > 0 {
			notifSQL := fmt.Sprintf(
				"UPDATE %s SET `version` = `version` + 1, `status` = '%s', `update_at` = %d WHERE `id` IN (%s)",
				tb, domain.SendStatusFailure.String(), now, m.successToString(),
			)
			sqls = append(sqls, notifSQL)
		}
	}

	if len(sqls) == 0 {
		sql := strings.Join(sqls, "; ")
		return tx.Exec(sql).Error
	}
	return nil
}

func (nd *NotifShardingDAO) GetById(ctx context.Context, id uint64) (Notification, error) {
	dst := nd.notifShardingStrategy.ShardWithId(id)
	db, ok := nd.dbs.Load(dst.DB)
	if !ok {
		return Notification{}, fmt.Errorf("failed to load db: %s", dst.DB)
	}

	var n Notification
	err := db.WithContext(ctx).Table(dst.Table).Where("id = ?", id).First(&n).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Notification{}, fmt.Errorf("%w: id = %d", errs.ErrNotificationNotFound, id)
		}
		return Notification{}, err
	}
	return n, nil
}

func (nd *NotifShardingDAO) GetByKey(ctx context.Context, bizId uint64, bizKey string) (Notification, error) {
	dst := nd.notifShardingStrategy.Shard(bizId, bizKey)
	db, ok := nd.dbs.Load(dst.DB)
	if !ok {
		return Notification{}, fmt.Errorf("failed to load db: %s", dst.DB)
	}

	var n Notification
	err := db.WithContext(ctx).Table(dst.Table).
		Where("`biz_id` = ? AND `biz_key` = ?", bizId, bizKey).
		First(&n).Error
	if err != nil {
		return Notification{}, fmt.Errorf("failed to get notification, bizId = %d, bizKey = %s, %w", bizId, bizKey, err)
	}
	return n, nil
}

func (nd *NotifShardingDAO) GetMapByIds(ctx context.Context, ids []uint64) (map[uint64]Notification, error) {
	idMap := make(map[[2]string][]uint64, len(ids))
	for _, id := range ids {
		dst := nd.notifShardingStrategy.ShardWithId(id)

		key := [2]string{dst.DB, dst.Table}
		val, ok := idMap[key]
		if ok {
			val = append(val, id)
		} else {
			val = []uint64{id}
		}

		idMap[key] = val
	}

	notifMap := make(map[uint64]Notification, len(idMap))
	mu := new(sync.RWMutex)

	// 广播查找
	var eg errgroup.Group
	for key, val := range idMap {
		mapKey := key
		mapVal := val
		eg.Go(func() error {
			var ns []Notification

			dbName := mapKey[0]
			db, ok := nd.dbs.Load(dbName)
			if !ok {
				return fmt.Errorf("failed to load db: %s", dbName)
			}
			tableName := mapKey[1]
			err := db.WithContext(ctx).Table(tableName).Where("id in (?)", mapVal).Find(&ns).Error
			for i := range ns {
				n := ns[i]
				mu.Lock()
				notifMap[n.Id] = n
				mu.Unlock()
			}
			return err
		})
	}
	return notifMap, eg.Wait()
}

func (nd *NotifShardingDAO) MarkSuccess(ctx context.Context, n Notification) error {
	now := time.Now().UnixMilli()
	dst := nd.notifShardingStrategy.ShardWithId(n.Id)
	cbLogDst := nd.cbLogShardingStrategy.ShardWithId(n.Id)

	db, ok := nd.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load db: %s", dst.DB)
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Table(dst.Table).Model(&Notification{}).Where("id = ?", n.Id).
			Updates(map[string]any{
				"status":    n.Status,
				"update_at": now,
				"version":   gorm.Expr("`version` + 1"),
			}).Error
		if err != nil {
			return err
		}

		// 标记 callback log 状态为 pending（可发送）
		return tx.Table(cbLogDst.Table).Model(&CallbackLog{}).Where("notification_id = ?", n.Id).
			Updates(map[string]any{
				"status":    domain.CallbackStatusPending,
				"update_at": now,
			}).Error
	})
}

func (nd *NotifShardingDAO) MarkFailure(ctx context.Context, n Notification) error {
	now := time.Now().UnixMilli()
	dst := nd.notifShardingStrategy.ShardWithId(n.Id)
	db, ok := nd.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load db: %s", dst.DB)
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Table(dst.Table).Model(&Notification{}).Where("id = ?", n.Id).
			Updates(map[string]any{
				"status":    n.Status,
				"update_at": now,
				"version":   gorm.Expr("`version` + 1"),
			}).Error
	})
}

func (nd *NotifShardingDAO) CompareAndSwapStatus(ctx context.Context, n Notification) error {
	dst := nd.notifShardingStrategy.ShardWithId(n.Id)
	db, ok := nd.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("failed to load db: %s", dst.DB)
	}

	res := db.WithContext(ctx).Table(dst.Table).
		Where("`id` = ? AND `version` = ?", n.Id, n.Version).
		Updates(map[string]any{
			"status":    n.Status,
			"version":   gorm.Expr("`version` + 1"),
			"update_at": time.Now().UnixMilli(),
		})
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected < 0 {
		return fmt.Errorf("%w: failed to concurrent competition", res.Error)
	}
	return nil
}

func (nd *NotifShardingDAO) FindReady(ctx context.Context, offset int, limit int) ([]Notification, error) {
	//TODO implement me
	panic("implement me")
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

type modifyIds struct {
	callbackTable string
	successIds    []uint64
	failureIds    []uint64
}

func (m *modifyIds) successToString() string {
	return m.listToString(m.successIds)
}

func (m *modifyIds) failureToString() string {
	return m.listToString(m.failureIds)
}

func (m *modifyIds) listToString(list []uint64) string {
	s := make([]string, len(list))
	for i := range list {
		s[i] = fmt.Sprintf("%d", list[i])
	}
	return strings.Join(s, ",")
}
