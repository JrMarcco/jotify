package jwt

import (
	"context"

	"github.com/JrMarcco/jotify/internal/errs"
)

const bizIdParamName = "biz_id"

type BizIdContextKey = struct{}

// ExtractBizId 从 context 中获取 biz id
func ExtractBizId(ctx context.Context) (uint64, error) {
	val := ctx.Value(BizIdContextKey{})
	if val == nil {
		return 0, errs.ErrBizIdNotFound
	}
	if val, ok := val.(uint64); ok {
		return val, nil
	}
	return 0, errs.ErrBizIdNotFound
}
