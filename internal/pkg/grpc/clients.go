package grpc

import (
	"fmt"
	"log"

	"github.com/JrMarcco/easy-kit/xsync"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients[T any] struct {
	etcdClient *clientv3.Client

	clientMap xsync.Map[string, T]
	creator   func(conn *grpc.ClientConn) T
}

func (c *Clients[T]) Get(serviceName string) T {
	client, ok := c.clientMap.Load(serviceName)
	if !ok {
		conn, err := c.createGrpcConn(serviceName)
		if err != nil {
			log.Panicf("failed to create grpc client for service %s: %v", serviceName, err)
		}

		client = c.creator(conn)
		c.clientMap.Store(serviceName, client)
	}
	return client
}

func (c *Clients[T]) createGrpcConn(serviceName string) (*grpc.ClientConn, error) {
	etcdResolver, err := resolver.NewBuilder(c.etcdClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd resolver: %w", err)
	}

	conn, err := grpc.NewClient(
		fmt.Sprintf("etcd:///%s", serviceName),
		grpc.WithResolvers(etcdResolver),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client: %w", err)
	}
	return conn, nil
}

func NewClients[T any](etcdClient *clientv3.Client, creator func(conn *grpc.ClientConn) T) *Clients[T] {
	return &Clients[T]{
		etcdClient: etcdClient,
		creator:    creator,
	}
}
