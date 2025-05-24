package retry

import (
	"fmt"
	"time"

	"github.com/JrMarcco/easy-kit/retry"
)

type Config struct {
	Type               string                    `json:"type"`
	FixedInterval      *FixedIntervalConfig      `json:"fixed_interval"`
	ExponentialBackoff *ExponentialBackoffConfig `json:"exponential_backoff"`
}

type ExponentialBackoffConfig struct {
	InitInterval time.Duration `json:"init_interval"`
	MaxInterval  time.Duration `json:"max_interval"`
	MaxTimes     int32         `json:"max_times"`
}

type FixedIntervalConfig struct {
	Interval time.Duration `json:"interval"`
	MaxTimes int32         `json:"max_times"`
}

func NewRetryStrategy(cfg Config) (retry.Strategy, error) {
	switch cfg.Type {
	case "fixed_interval":
		return retry.NewFixedIntervalStrategy(cfg.FixedInterval.Interval, cfg.FixedInterval.MaxTimes)
	case "exponential_backoff":
		return retry.NewExponentialBackoffStrategy(
			cfg.ExponentialBackoff.InitInterval,
			cfg.ExponentialBackoff.MaxInterval,
			cfg.ExponentialBackoff.MaxTimes,
		)
	default:
		return nil, fmt.Errorf("unknown retry strategy type: %s", cfg.Type)
	}
}
