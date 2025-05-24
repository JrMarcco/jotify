package conf

import "github.com/JrMarcco/jotify/internal/repository"

//go:generate mockgen -source=./biz_conf.go -destination=./mock/biz_conf.mock.go -pack=confmock -type=BizConfService
type BizConfService interface {
}

var _ BizConfService = (*DefaultBizConfService)(nil)

type DefaultBizConfService struct {
	bcRepo repository.BizConfRepo
}

func NewDefaultBizConfService(bcRepo repository.BizConfRepo) *DefaultBizConfService {
	return &DefaultBizConfService{
		bcRepo: bcRepo,
	}
}
