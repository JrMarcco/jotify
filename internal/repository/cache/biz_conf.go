package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/JrMarcco/jotify/internal/domain"
)

const (
	BizConfPrefix  = "biz_conf"
	DefaultExpires = 15 * time.Minute
)

type BizConfCache interface {
	Get(ctx context.Context, id uint64) (domain.BizConf, error)
	Set(ctx context.Context, id uint64, conf domain.BizConf) error
}

func BizConfKey(bizId uint64) string {
	return fmt.Sprintf("%s:%d", BizConfPrefix, bizId)
}
