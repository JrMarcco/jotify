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
	TxNotifConf  xsql.JsonColumn[domain.TxNotifConf]
	RateLimit    int32
	QuotaConf    xsql.JsonColumn[domain.QuotaConf]
	CallbackConf xsql.JsonColumn[domain.CallbackConf]
	CreatedAt    int64
	UpdatedAt    int64
}

func (bc BizConf) TableName() string {
	return "biz_conf"
}

type BizConfDAO interface {
	GetById(ctx context.Context, id uint64) (BizConf, error)
}

var _ BizConfDAO = (*DefaultBizConfDAO)(nil)

type DefaultBizConfDAO struct {
	db *gorm.DB
}

func (d *DefaultBizConfDAO) GetById(ctx context.Context, id uint64) (BizConf, error) {
	var bizConf BizConf

	err := d.db.WithContext(ctx).Where("id = ?", id).First(&bizConf).Error
	if err != nil {
		return BizConf{}, err
	}
	return bizConf, nil
}

func NewDefaultBizConfDAO(db *gorm.DB) *DefaultBizConfDAO {
	return &DefaultBizConfDAO{
		db: db,
	}
}
