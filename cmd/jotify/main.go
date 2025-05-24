package main

import (
	"github.com/JrMarcco/jotify/internal/ioc"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	initViper()

	fx.New(
		// 初始化 zap.Logger
		ioc.LoggerFxOpt,

		// 初始化雪花算法 id 生成器
		ioc.IdFxOpt,

		// 初始化数据库
		ioc.DBFxOpt,

		// 初始化 etcd
		ioc.EtcdFxOpt,

		// 初始化 Repo
		ioc.RepoFxOpt,

		// 初始化 Service
		ioc.ServiceFxOpt,

		// 初始化注册中心
		ioc.RegistryFxOpt,
		// 初始化 grpc.Server
		ioc.GrpcFxOpt,

		// 初始化 ioc.App
		ioc.AppFxOpt,

		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		}),

		// 实际运行方法，即调用 ioc.AppLifecycle 方法
		ioc.AppFxInvoke,
		// 确保日志缓冲区被刷新
		ioc.LoggerFxInvoke,
	).Run()
}

// initViper 初始化 viper
func initViper() {
	configFile := pflag.String("config", "etc/config.yaml", "配置文件路径")
	pflag.Parse()

	viper.SetConfigFile(*configFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}
