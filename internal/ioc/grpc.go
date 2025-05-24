package ioc

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"

	notificationv1 "github.com/JrMarcco/jotify-api/api/notification/v1"
	grpcapi "github.com/JrMarcco/jotify/internal/api/grpc"
	"github.com/JrMarcco/jotify/internal/api/grpc/interceptor/jwt"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

var GrpcFxOpt = fx.Provide(
	InitGrpcServer,
	grpcapi.NewNotificationServer,
)

func InitGrpcServer(server *grpcapi.NotificationServer) *grpc.Server {
	type Config struct {
		PriPem string `mapstructure:"private"`
		PubPem string `mapstructure:"public"`
	}

	cfg := &Config{}
	if err := viper.UnmarshalKey("jwt", cfg); err != nil {
		panic(err)
	}
	priKey, pubKey := loadJwtKeypair(cfg.PriPem, cfg.PubPem)

	grpcSvr := grpc.NewServer(
		// 注册拦截器
		grpc.UnaryInterceptor(InterceptorOf(
			jwt.Builder(priKey, pubKey).Build(),
		)),
	)
	notificationv1.RegisterNotificationServiceServer(grpcSvr, server)
	notificationv1.RegisterNotificationQueryServiceServer(grpcSvr, server)

	return grpcSvr
}

// loadJwtKeypair 加载 jwt 密钥对。
//
// PEM 块本身标注的是密钥对，而不是具体的 ed25519 密钥对。
// 所有标准公钥格式都需要先由 x509 包处理进行转换后类型断言才能获得 ed25519 密钥对。
func loadJwtKeypair(priPem, pubPem string) (ed25519.PrivateKey, ed25519.PublicKey) {
	priKeyBlock, _ := pem.Decode([]byte(priPem))
	if priKeyBlock == nil {
		panic("failed to decode private key PEM")
	}
	priKey, err := x509.ParsePKCS8PrivateKey(priKeyBlock.Bytes)
	if err != nil {
		panic(err)
	}

	pubKeyBlock, _ := pem.Decode([]byte(pubPem))
	if pubKeyBlock == nil {
		panic("failed to decode public key PEM")
	}
	publicKey, err := x509.ParsePKIXPublicKey(pubKeyBlock.Bytes)
	if err != nil {
		panic(err)
	}

	return priKey.(ed25519.PrivateKey), publicKey.(ed25519.PublicKey)
}

// InterceptorOf 自定义拦截器链，grpc 官方只允许一次 grpc.UnaryInterceptor 调用
func InterceptorOf(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// 顺序嵌套调用
		chainedHandler := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			thisInterceptor := interceptors[i]
			next := chainedHandler
			chainedHandler = func(ctx context.Context, req any) (any, error) {
				return thisInterceptor(ctx, req, info, next)
			}
		}
		return chainedHandler(ctx, req)
	}
}
