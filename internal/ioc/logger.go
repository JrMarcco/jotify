package ioc

import (
	"context"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var LoggerFxOpt = fx.Provide(
	InitLogger,
)

var LoggerFxInvoke = fx.Invoke(
	LoggerLifecycle,
)

func InitLogger() *zap.Logger {
	type config struct {
		Env string `mapstructure:"env"`
	}

	cfg := &config{}

	err := viper.UnmarshalKey("profile", cfg)
	if err != nil {
		panic(err)
	}

	var zLogger *zap.Logger
	switch cfg.Env {
	case "prod":
		zLogger, err = zap.NewProduction()
	default:
		zLogger, err = zap.NewDevelopment()
	}
	if err != nil {
		panic(err)
	}
	return zLogger
}

func LoggerLifecycle(lc fx.Lifecycle, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			_ = logger.Sync()
			return nil
		},
	})
}
