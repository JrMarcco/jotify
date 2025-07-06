package client

import "context"

const (
	AttrWeight      = "attr_weight"
	AttrReadWeight  = "attr_read_weight"
	AttrWriteWeight = "attr_write_weight"
	AttrGroup       = "attr_group"
	AttrNode        = "attr_node"
)

type ContextKeyGroup struct{}

// WithGroup 在 context.Context 内写入 group 信息
func WithGroup(ctx context.Context, group string) context.Context {
	return context.WithValue(ctx, ContextKeyGroup{}, group)
}

type ContextKeyReqType struct{}

// WithReqType 在 context.Context 内写入 request type 信息
// read request  = 0
// write request = 1
func WithReqType(ctx context.Context, reqType uint8) context.Context {
	return context.WithValue(ctx, ContextKeyReqType{}, reqType)
}
