package registry

import (
	"context"
	"io"
)

type Registry interface {
	Register(ctx context.Context, si ServiceInstance) error
	Unregister(ctx context.Context, si ServiceInstance) error
	ListService(ctx context.Context, serviceName string) ([]ServiceInstance, error)
	Subscribe(serviceName string) <-chan struct{}

	io.Closer
}

type ServiceInstance struct {
	Name        string
	Addr        string
	Group       string
	ReadWeight  int
	WriteWeight int
}
