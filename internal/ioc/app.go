package ioc

import (
	"context"
	"net"
	"time"

	"github.com/JrMarcco/jotify/internal/pkg/registry"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var AppFxOpt = fx.Provide(
	InitApp,
)

var AppFxInvoke = fx.Invoke(
	AppLifecycle,
)

type App struct {
	*grpc.Server

	timeout         time.Duration
	registry        registry.Registry
	serviceInstance registry.ServiceInstance

	logger *zap.Logger
}

func (app *App) Start() error {
	ln, err := net.Listen("tcp", app.serviceInstance.Addr)
	if err != nil {
		return err
	}

	// 启动 gRPC 服务器
	go func() {
		if serveErr := app.Serve(ln); err != nil {
			panic(serveErr)
		}
	}()

	// 注册服务到注册中心
	if app.registry != nil {
		registerCtx, cancel := context.WithTimeout(context.Background(), app.timeout)
		regErr := app.registry.Register(registerCtx, app.serviceInstance)
		cancel()

		if regErr != nil {
			return regErr
		}
	}
	return nil
}

func (app *App) Close() {
	if app.registry != nil {
		// 从注册中心注销服务
		unregisterCtx, cancel := context.WithTimeout(context.Background(), app.timeout)
		err := app.registry.Unregister(unregisterCtx, app.serviceInstance)
		cancel()

		if err != nil {
			// 记录错误但不返回，确保服务器能够正常关闭
			app.logger.Error("[jotify] unregister service failed", zap.Error(err))
		}

		_ = app.registry.Close()
	}

	// 优雅退出
	app.GracefulStop()
}

func InitApp(grpcServer *grpc.Server, r registry.Registry, zLogger *zap.Logger) *App {
	type config struct {
		Name        string `mapstructure:"name"`
		Addr        string `mapstructure:"addr"`
		Group       string `mapstructure:"group"`
		ReadWeight  int    `mapstructure:"read_weight"`
		WriteWeight int    `mapstructure:"write_weight"`
		Timeout     int    `mapstructure:"timeout"`
	}
	cfg := &config{}
	if err := viper.UnmarshalKey("app", cfg); err != nil {
		panic(err)
	}

	serviceInstance := registry.ServiceInstance{
		Name:        cfg.Name,
		Addr:        cfg.Addr,
		Group:       cfg.Group,
		WriteWeight: cfg.WriteWeight,
		ReadWeight:  cfg.ReadWeight,
	}

	return &App{
		Server:          grpcServer,
		timeout:         time.Duration(cfg.Timeout) * time.Millisecond,
		registry:        r,
		serviceInstance: serviceInstance,
		logger:          zLogger,
	}
}

func AppLifecycle(lc fx.Lifecycle, app *App) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return app.Start()
		},
		OnStop: func(ctx context.Context) error {
			app.Close()
			return nil
		},
	})
}
