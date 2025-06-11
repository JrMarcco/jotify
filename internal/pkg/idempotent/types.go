package idempotent

import "context"

// Strategy 幂等策略
type Strategy interface {
	Exists(ctx context.Context, key string) (bool, error)
	MultiExists(ctx context.Context, keys []string) (map[string]bool, error)
}
