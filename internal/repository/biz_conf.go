package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/JrMarcco/jotify/internal/domain"
	"github.com/JrMarcco/jotify/internal/errs"
	"github.com/JrMarcco/jotify/internal/repository/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BizConfRepo interface {
	GetById(ctx context.Context, id uint64) (domain.BizConf, error)
}

var _ BizConfRepo = (*DefaultBizConfRepo)(nil)

type DefaultBizConfRepo struct {
	bizConfDAO dao.BizConfDAO
	logger     *zap.Logger
}

func (d *DefaultBizConfRepo) GetById(ctx context.Context, id uint64) (domain.BizConf, error) {
	if id <= 0 {
		return domain.BizConf{}, fmt.Errorf("%w", errs.ErrInvalidParam)
	}

	conf, err := d.bizConfDAO.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.BizConf{}, fmt.Errorf("%w", errs.ErrBizIdNotFound)
		}
		return domain.BizConf{}, err
	}
	return conf, nil
}

func NewDefaultBizConfRepo(logger *zap.Logger) *DefaultBizConfRepo {
	return &DefaultBizConfRepo{
		logger: logger,
	}
}
