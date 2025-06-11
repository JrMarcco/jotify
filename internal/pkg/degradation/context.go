package degradation

import "context"

type degradeContext struct{}

func WithDegrade(ctx context.Context) context.Context {
	return context.WithValue(ctx, degradeContext{}, true)
}

func ShouldDegrade(ctx context.Context) bool {
	return ctx.Value(degradeContext{}) == true
}
