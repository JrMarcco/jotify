package cache

import (
	"context"

	"github.com/JrMarcco/jotify/internal/domain"
)

type QuotaCache interface {
	Incr(ctx context.Context, param QuotaParam) error
	BatchIncr(ctx context.Context, params []QuotaParam) error
	Decr(ctx context.Context, param QuotaParam) error
	BatchDecr(ctx context.Context, params []QuotaParam) error
}

type QuotaParam struct {
	BizId   uint64
	Quota   int32
	Channel domain.Channel
}
