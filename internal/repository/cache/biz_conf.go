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

var (
	ErrBizConfCacheKeyNotFound = fmt.Errorf("[jotify] biz conf cache key not found")
	ErrWrongTypeOfBizConfCache = fmt.Errorf("[jotify] wrong type of biz conf cache")
)

type BizConfCache interface {
	Set(ctx context.Context, id uint64, conf domain.BizConf) error
	Get(ctx context.Context, id uint64) (domain.BizConf, error)
}

func BizConfCacheKey(bizId uint64) string {
	return fmt.Sprintf("%s:%d", BizConfPrefix, bizId)
}
