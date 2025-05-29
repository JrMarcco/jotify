package repository

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository/dao"
)

type ChannelTplRepo interface {
	GetById(ctx context.Context, id uint64) (domain.ChannelTpl, error)
	GetVersionByVersionId(ctx context.Context, id uint64) (domain.ChannelTplVersion, error)
}

var _ ChannelTplRepo = (*DefaultChannelTplRepo)(nil)

type DefaultChannelTplRepo struct {
	tplDAO dao.ChannelTplDAO
}

func (d *DefaultChannelTplRepo) GetById(ctx context.Context, id uint64) (domain.ChannelTpl, error) {
	entity, err := d.tplDAO.GetById(ctx, id)
	if err != nil {
		return domain.ChannelTpl{}, err
	}
	return d.toDomainTemplate(entity), nil
}

func (d *DefaultChannelTplRepo) GetVersionByVersionId(ctx context.Context, versionId uint64) (domain.ChannelTplVersion, error) {
	version, err := d.tplDAO.GetVersionsById(ctx, versionId)
	if err != nil {
		return domain.ChannelTplVersion{}, err
	}
	return d.toDomainVersion(version), nil
}

func (d *DefaultChannelTplRepo) toDomainVersion(entity dao.ChannelTplVersion) domain.ChannelTplVersion {
	return domain.ChannelTplVersion{
		Id:           entity.Id,
		ChannelTplId: entity.ChannelTplId,
		Name:         entity.Name,
		Signature:    entity.Signature,
		Content:      entity.Content,
		Remark:       entity.Remark,
		AuditId:      entity.AuditId,
		AuditorId:    entity.AuditorId,
		AuditStatus:  domain.AuditStatus(entity.AuditStatus),
		AuditAt:      entity.AuditAt,
		RejectReason: entity.RejectReason,
		LastReviewAt: entity.LastReviewAt,
		CreateAt:     entity.CreatedAt,
		UpdateAt:     entity.UpdatedAt,
	}
}

func (d *DefaultChannelTplRepo) toDomainTemplate(entity dao.ChannelTpl) domain.ChannelTpl {
	return domain.ChannelTpl{
		Id:                 entity.Id,
		OwnerId:            entity.OwnerId,
		OwnerType:          domain.OwnerType(entity.OwnerType),
		Name:               entity.Name,
		Description:        entity.Description,
		Channel:            domain.Channel(entity.Channel),
		BizType:            domain.BizType(entity.BizType),
		ActivatedVersionId: entity.ActivatedVersionId,
		CreateAt:           entity.CreatedAt,
		UpdateAt:           entity.UpdatedAt,
	}
}

func NewDefaultChannelTplRepo(dao dao.ChannelTplDAO) *DefaultChannelTplRepo {
	return &DefaultChannelTplRepo{
		tplDAO: dao,
	}
}
