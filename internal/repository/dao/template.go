package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/JrMarcco/jotify/internal/errs"
	"gorm.io/gorm"
)

// ChannelTpl 渠道模板
type ChannelTpl struct {
	Id                 uint64
	OwnerId            uint64
	OwnerType          string
	Name               string
	Description        string
	Channel            string
	BizType            string
	ActivatedVersionId uint64
	CreatedAt          int64
	UpdatedAt          int64
}

func (ct ChannelTpl) TableName() string {
	return "channel_tpl"
}

type ChannelTplVersion struct {
	Id           uint64
	ChannelTplId uint64
	Name         string
	Signature    string
	Content      string
	Remark       string
	AuditId      uint64
	AuditorId    uint64
	AuditAt      int64
	AuditStatus  string
	RejectReason string
	LastReviewAt int64
	CreatedAt    int64
	UpdatedAt    int64
}

func (c ChannelTplVersion) TableName() string {
	return "channel_tpl_version"
}

type ChannelTplProvider struct {
	Id              uint64
	TplId           uint64
	TplVersionId    uint64
	ProviderId      uint64
	ProviderName    string
	ProviderChannel string
	RequestId       string
	ProviderTplId   string
	AuditStatus     string
	RejectReason    string
	LastReviewAt    int64
	CreatedAt       int64
	UpdatedAt       int64
}

func (c ChannelTplProvider) TableName() string {
	return "channel_tpl_provider"
}

var _ ChannelTplDAO = (*DefaultChannelTplDAO)(nil)

type ChannelTplDAO interface {
	GetById(ctx context.Context, id uint64) (ChannelTpl, error)
	GetVersionsById(ctx context.Context, versionId uint64) (ChannelTplVersion, error)
	GetVersionsByIds(ctx context.Context, versionIds []uint64) ([]ChannelTplVersion, error)
}

type DefaultChannelTplDAO struct {
	db *gorm.DB
}

func (d *DefaultChannelTplDAO) GetById(ctx context.Context, id uint64) (ChannelTpl, error) {
	var tpl ChannelTpl
	if err := d.db.WithContext(ctx).Where("id = ?", id).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ChannelTpl{}, fmt.Errorf("%w", errs.ErrChannelTplNotFound)
		}
		return ChannelTpl{}, err
	}
	return tpl, nil
}

func (d *DefaultChannelTplDAO) GetVersionsById(ctx context.Context, versionId uint64) (ChannelTplVersion, error) {
	var version ChannelTplVersion
	if err := d.db.WithContext(ctx).Where("id = ?", versionId).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ChannelTplVersion{}, fmt.Errorf("%w", errs.ErrChannelTplVersionNotFound)
		}
		return ChannelTplVersion{}, err
	}
	return version, nil
}

func (d *DefaultChannelTplDAO) GetVersionsByIds(ctx context.Context, versionIds []uint64) ([]ChannelTplVersion, error) {
	if len(versionIds) == 0 {
		return []ChannelTplVersion{}, nil
	}

	var versions []ChannelTplVersion
	res := d.db.WithContext(ctx).Where("channel_tpl_id IN (?)", versionIds).Find(&versions)
	if res.Error != nil {
		return nil, res.Error
	}
	return versions, nil
}

func NewDefaultChannelTplDAO(db *gorm.DB) *DefaultChannelTplDAO {
	return &DefaultChannelTplDAO{
		db: db,
	}
}
