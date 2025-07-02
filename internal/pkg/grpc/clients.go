package grpc

import (
	"fmt"
	"log"

	"github.com/JrMarcco/easy-kit/xsync"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

type Clients[T any] struct {
	clientMap xsync.Map[string, T]

	rb       resolver.Builder
	bb       balancer.Builder
	insecure bool

	creator func(conn *grpc.ClientConn) T
}

func (c *Clients[T]) Get(serviceName string) T {
	client, ok := c.clientMap.Load(serviceName)
	if !ok {
		conn, err := c.dial(serviceName)
		if err != nil {
			log.Panicf("failed to create grpc client for service %s: %v", serviceName, err)
		}

		client = c.creator(conn)
		c.clientMap.Store(serviceName, client)
	}
	return client
}

func (c *Clients[T]) dial(serviceName string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithResolvers(c.rb),
		grpc.WithNoProxy(),
	}

	if c.insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if c.bb != nil {
		opts = append(opts, grpc.WithDefaultServiceConfig(
			fmt.Sprintf(`{"loadBalancingPolicy: %q"}`, c.bb.Name()),
		))
	}
	// "registry:///%s" 的 registry 对应 grpc resolver 的 scheme
	addr := fmt.Sprintf("registry:///%s", serviceName)
	return grpc.NewClient(addr, opts...)
}

func NewClients[T any](rb resolver.Builder, bb balancer.Builder, creator func(conn *grpc.ClientConn) T) *Clients[T] {
	return &Clients[T]{
		rb:      rb,
		bb:      bb,
		creator: creator,
	}
}
