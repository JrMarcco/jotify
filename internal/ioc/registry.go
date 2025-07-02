package ioc

import (
	"github.com/JrMarcco/jotify/internal/pkg/registry"
	"github.com/JrMarcco/jotify/internal/pkg/registry/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"
)

var RegistryFxOpt = fx.Provide(
	fx.Annotate(
		InitRegistry,
		fx.As(new(registry.Registry)),
	),
)

func InitRegistry(client *clientv3.Client) *etcd.Registry {
	r, err := etcd.NewRegistry(client)
	if err != nil {
		panic(err)
	}
	return r
}
