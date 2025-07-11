package ioc

import (
	"github.com/JrMarcco/dlock"
	dr "github.com/JrMarcco/dlock/redis"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var RedisFxOpt = fx.Provide(
	InitRedis,
	InitDClient,
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

// InitDClient 初始化分布式锁客户端
func InitDClient(rc redis.Cmdable) dlock.Dclient {
	return dr.NewDClientBuilder(rc).Build()
}
