package sharding

import (
	"context"
)

// Strategy 分库分表策略
type Strategy interface {
	Shard(bizId uint64, bizKey string) Dst
	ShardWithId(id uint64) Dst
	BroadCast() []Dst
}

// Dst 目标信息，包含分库和分表信息
type Dst struct {
	DBSuffix    uint64
	TableSuffix uint64

	DB    string
	Table string
}

type dstContextKey struct{}

func ContextWitDst(ctx context.Context, dst Dst) context.Context {
	return context.WithValue(ctx, dstContextKey{}, dst)
}

func DstFromContext(ctx context.Context) (Dst, bool) {
	val := ctx.Value(dstContextKey{})
	dst, ok := val.(Dst)
	return dst, ok
}
