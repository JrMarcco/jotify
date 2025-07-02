package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/JrMarcco/jotify/internal/pkg/registry"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var typeMap = map[mvccpb.Event_EventType]registry.EventType{
	mvccpb.PUT:    registry.EventTypePut,
	mvccpb.DELETE: registry.EventTypeDel,
}

var _ registry.Registry = (*Registry)(nil)

type Registry struct {
	mu sync.Mutex

	etcdClient  *clientv3.Client
	etcdSession *concurrency.Session
	watchCancel []context.CancelFunc
}

func (r *Registry) Register(ctx context.Context, si registry.ServiceInstance) error {
	val, err := json.Marshal(si)
	if err != nil {
		return err
	}
	_, err = r.etcdClient.Put(ctx, r.instanceKey(si), string(val), clientv3.WithLease(r.etcdSession.Lease()))
	return err
}

func (r *Registry) Unregister(ctx context.Context, si registry.ServiceInstance) error {
	_, err := r.etcdClient.Delete(ctx, r.instanceKey(si))
	return err
}

func (r *Registry) ListService(ctx context.Context, serviceName string) ([]registry.ServiceInstance, error) {
	resp, err := r.etcdClient.Get(ctx, r.serviceKey(serviceName), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	res := make([]registry.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var si registry.ServiceInstance
		if err := json.Unmarshal(kv.Value, &si); err != nil {
			return nil, err
		}
		res = append(res, si)
	}
	return res, nil
}

func (r *Registry) Subscribe(serviceName string) <-chan registry.Event {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = clientv3.WithRequireLeader(ctx)

	r.mu.Lock()
	r.watchCancel = append(r.watchCancel, cancel)
	r.mu.Unlock()

	ch := r.etcdClient.Watch(ctx, r.serviceKey(serviceName), clientv3.WithPrefix())
	res := make(chan registry.Event)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case resp := <-ch:
				if resp.Canceled {
					return
				}

				if resp.Err() != nil {
					continue
				}

				for _, e := range resp.Events {
					// 事件类型转换：mvccpb.Event_EventType -> registry.EventType
					res <- registry.Event{
						Type: typeMap[e.Type],
					}
				}
			}
		}
	}()
	return res
}

func (r *Registry) serviceKey(serviceName string) string {
	return fmt.Sprintf("/notification/%s", serviceName)
}

func (r *Registry) instanceKey(si registry.ServiceInstance) string {
	return fmt.Sprintf("/notification/%s/%s", si.Name, si.Addr)
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, cancel := range r.watchCancel {
		cancel()
	}
	return r.etcdSession.Close()
}

func NewRegistry(client *clientv3.Client) (*Registry, error) {
	session, err := concurrency.NewSession(client)
	if err != nil {
		return nil, err
	}
	return &Registry{
		etcdSession: session,
		etcdClient:  client,
	}, nil
}
