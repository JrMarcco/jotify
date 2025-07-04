package resolver

import (
	"context"
	"time"

	"github.com/JrMarcco/jotify/internal/pkg/client"
	"github.com/JrMarcco/jotify/internal/pkg/registry"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

var _ resolver.Builder = (*GrpcResolverBuilder)(nil)

type GrpcResolverBuilder struct {
	registry registry.Registry
	timeout  time.Duration
}

func (b *GrpcResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	r := &GrpcResolver{
		registry: b.registry,
		timeout:  b.timeout,
		target:   target,
		cc:       cc,
		ch:       make(chan struct{}),
	}

	r.resolve()
	go r.watch()
	return r, nil
}

func (b *GrpcResolverBuilder) Scheme() string {
	return "registry"
}

func NewGrpcResolverBuilder(r registry.Registry, timeout time.Duration) *GrpcResolverBuilder {
	return &GrpcResolverBuilder{
		registry: r,
		timeout:  timeout,
	}
}

var _ resolver.Resolver = (*GrpcResolver)(nil)

type GrpcResolver struct {
	registry registry.Registry
	timeout  time.Duration

	target resolver.Target
	cc     resolver.ClientConn

	ch chan struct{} // 用来控制 watch 方法退出
}

func (r *GrpcResolver) ResolveNow(_ resolver.ResolveNowOptions) {
	r.resolve()
}

func (r *GrpcResolver) resolve() {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	instances, err := r.registry.ListService(ctx, r.target.Endpoint())
	cancel()

	if err != nil {
		r.cc.ReportError(err)
		return
	}

	addrs := make([]resolver.Address, 0, len(instances))
	for _, instance := range instances {
		addrs = append(addrs, resolver.Address{
			Addr:       instance.Addr,
			ServerName: instance.Name,
			Attributes: attributes.New(client.AttrReadWeight, instance.ReadWeight).
				WithValue(client.AttrWriteWeight, instance.WriteWeight).
				WithValue(client.AttrGroup, instance.Group).
				WithValue(client.AttrNode, instance.Name),
		})
	}

	err = r.cc.UpdateState(resolver.State{Addresses: addrs})
	if err != nil {
		r.cc.ReportError(err)
		return
	}
}

func (r *GrpcResolver) Close() {
	close(r.ch)
}

func (r *GrpcResolver) watch() {
	events := r.registry.Subscribe(r.target.Endpoint())

	for {
		select {
		case <-events:
			r.resolve()
		case <-r.ch:
			return
		}
	}
}
