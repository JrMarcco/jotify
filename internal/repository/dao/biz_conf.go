package dao

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/pkg/xsql"
	"gorm.io/gorm"
)

type BizConf struct {
	Id           uint64
	OwnerId      uint64
	OwnerType    string
	ChannelConf  xsql.JsonColumn[domain.ChannelConf]
	txNotifConf  xsql.JsonColumn[domain.TxNotifConf]
	RateLimit    int
	Quota        xsql.JsonColumn[domain.Quota]
	CallbackConf xsql.JsonColumn[domain.CallbackConf]
	CreatedAt    int64
	UpdatedAt    int64
}

func (bc BizConf) TableName() string {
	return "biz_conf"
}

type BizConfDAO interface {
	GetById(ctx context.Context, id uint64) (domain.BizConf, error)
}

var _ BizConfDAO = (*DefaultBizConfDAO)(nil)

type DefaultBizConfDAO struct {
	db *gorm.DB
}

func (d DefaultBizConfDAO) GetById(ctx context.Context, id uint64) (domain.BizConf, error) {
	// 先从本地缓存获取
	// TODO implement me
	panic("implement me")
}

func NewDefaultBizConfDAO(db *gorm.DB) *DefaultBizConfDAO {
	return &DefaultBizConfDAO{
		db: db,
	}
}
