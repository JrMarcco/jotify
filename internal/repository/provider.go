package repository

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/repository/dao"
)

type ProviderRepo interface {
	GetByNameAndTplInfo(ctx context.Context, name string, tplId uint64, tplVersionId uint64, tplChannel string) ([]domain.ChannelTplProvider, error)
}

var _ ProviderRepo = (*DefaultProviderRepo)(nil)

type DefaultProviderRepo struct {
	providerDAO dao.ProviderDAO
}

func (d *DefaultProviderRepo) GetByNameAndTplInfo(ctx context.Context, name string, tplId uint64, tplVersionId uint64, tplChannel string) ([]domain.ChannelTplProvider, error) {
	providers, err := d.providerDAO.GetByNameAndTplInfo(ctx, name, tplId, tplVersionId, tplChannel)
	if err != nil {
		return nil, err
	}

	res := make([]domain.ChannelTplProvider, 0, len(providers))
	for _, provider := range providers {
		res = append(res, d.toDomainChannelTplProvider(provider))
	}
	return res, nil
}

func (d *DefaultProviderRepo) toDomainChannelTplProvider(entity dao.ChannelTplProvider) domain.ChannelTplProvider {
	return domain.ChannelTplProvider{
		Id:              entity.Id,
		TplId:           entity.TplId,
		TplVersionId:    entity.TplVersionId,
		ProviderId:      entity.ProviderId,
		ProviderName:    entity.ProviderName,
		ProviderChannel: domain.Channel(entity.ProviderChannel),
		RequestId:       entity.RequestId,
		ProviderTplId:   entity.ProviderTplId,
		AuditStatus:     domain.AuditStatus(entity.AuditStatus),
		RejectReason:    entity.RejectReason,
		LastReviewAt:    entity.LastReviewAt,
		CreateAt:        entity.CreatedAt,
		UpdateAt:        entity.UpdatedAt,
	}
}

func NewDefaultProviderRepo(providerDAO dao.ProviderDAO) *DefaultProviderRepo {
	return &DefaultProviderRepo{
		providerDAO: providerDAO,
	}
}
