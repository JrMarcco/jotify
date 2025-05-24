package ioc

import (
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
)

var EtcdFxOpt = fx.Provide(
	InitEtcdClient,
)

func InitEtcdClient() *clientv3.Client {
	type config struct {
		Username  string   `mapstructure:"username"`
		Password  string   `mapstructure:"password"`
		Endpoints []string `mapstructure:"endpoints"`
	}
	cfg := &config{}
	if err := viper.UnmarshalKey("etcd", cfg); err != nil {
		panic(err)
	}

	client, err := clientv3.New(clientv3.Config{
		Username:  cfg.Username,
		Password:  cfg.Password,
		Endpoints: cfg.Endpoints,
	})
	if err != nil {
		panic(err)
	}
	return client
}
