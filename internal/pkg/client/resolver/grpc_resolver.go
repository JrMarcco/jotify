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
		close:    make(chan struct{}),
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

	close chan struct{}
}

func (r *GrpcResolver) ResolveNow(_ resolver.ResolveNowOptions) {
	r.resolve()
}

func (r *GrpcResolver) resolve() {
	serviceName := r.target.Endpoint()
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	instances, err := r.registry.ListService(ctx, serviceName)
	cancel()

	if err != nil {
		r.cc.ReportError(err)
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
	}
}

func (r *GrpcResolver) Close() {
	r.close <- struct{}{}
}

func (r *GrpcResolver) watch() {
	events := r.registry.Subscribe(r.target.Endpoint())

	for {
		select {
		case <-events:
			r.resolve()
		case <-r.close:
			return
		}
	}
}
