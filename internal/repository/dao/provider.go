package dao

import (
	"context"

	"gorm.io/gorm"
)

type Provider struct {
	Id      uint64
	Name    string
	Channel string

	Endpoint  string
	RegionId  string
	AppId     string
	ApiKey    string
	ApiSecret string

	Weight      int32
	QpsLimit    int32
	DailyLimit  int32
	CallbackUrl string

	Status    string
	CreatedAt int64
	UpdatedAt int64
}

func (p Provider) TableName() string {
	return "provider"
}

type ProviderDAO interface {
	GetByNameAndTplInfo(ctx context.Context, name string, tplId uint64, tplVersionId uint64, tplChannel string) ([]ChannelTplProvider, error)
}

var _ ProviderDAO = (*DefaultProviderDAO)(nil)

type DefaultProviderDAO struct {
	db *gorm.DB
}

// GetByNameAndTplInfo 根据名称以及模板信息获取已通过审核的供应商信息。
//
// 模板信息包含模板id、模板版本id、模板渠道。
func (d *DefaultProviderDAO) GetByNameAndTplInfo(ctx context.Context, name string, tplId uint64, tplVersionId uint64, tplChannel string) ([]ChannelTplProvider, error) {
	var providers []ChannelTplProvider
	err := d.db.WithContext(ctx).Model(&ChannelTplProvider{}).
		Where("provider_name = ? AND tpl_id = ? AND tpl_version_id = ? AND tpl_channel = ?", name, tplId, tplVersionId, tplChannel).
		Find(&providers).Error
	return providers, err
}

func NewDefaultProviderDAO(db *gorm.DB) *DefaultProviderDAO {
	return &DefaultProviderDAO{
		db: db,
	}
}
