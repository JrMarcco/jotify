package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var RedisFxOpt = fx.Provide(
	InitRedis,
)

func InitRedis() redis.Cmdable {
	type config struct {
		Addr     string
		Password string
	}
	cfg := &config{}
	if err := viper.UnmarshalKey("redis", cfg); err != nil {
		panic(err)
	}
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
	})
}
