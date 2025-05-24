package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var typeMap = map[mvccpb.Event_EventType]EventType{
	mvccpb.PUT:    EventTypePut,
	mvccpb.DELETE: EventTypeDelete,
}

var _ Registry = (*EtcdRegistry)(nil)

type EtcdRegistry struct {
	mu sync.Mutex

	client      *clientv3.Client
	session     *concurrency.Session
	watchCancel []context.CancelFunc
}

func (r *EtcdRegistry) Register(ctx context.Context, si ServiceInstance) error {
	val, err := json.Marshal(si)
	if err != nil {
		return err
	}
	_, err = r.client.Put(ctx, r.siKey(si), string(val), clientv3.WithLease(r.session.Lease()))
	return err
}

func (r *EtcdRegistry) Unregister(ctx context.Context, si ServiceInstance) error {
	_, err := r.client.Delete(ctx, r.siKey(si))
	return err
}

func (r *EtcdRegistry) siKey(si ServiceInstance) string {
	return fmt.Sprintf("/notification/%s/%s", si.Name, si.Addr)
}

func (r *EtcdRegistry) ListService(ctx context.Context, serviceName string) ([]ServiceInstance, error) {
	resp, err := r.client.Get(ctx, r.serviceKey(serviceName), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	res := make([]ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var si ServiceInstance
		if err := json.Unmarshal(kv.Value, &si); err != nil {
			return nil, err
		}
		res = append(res, si)
	}
	return res, nil
}

func (r *EtcdRegistry) Subscribe(serviceName string) <-chan Event {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = clientv3.WithRequireLeader(ctx)

	r.mu.Lock()
	r.watchCancel = append(r.watchCancel, cancel)
	r.mu.Unlock()

	ch := r.client.Watch(ctx, r.serviceKey(serviceName), clientv3.WithPrefix())
	res := make(chan Event)

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
					res <- Event{
						Type: typeMap[e.Type],
					}
				}
			}
		}
	}()
	return res
}

func (r *EtcdRegistry) serviceKey(serviceName string) string {
	return fmt.Sprintf("/notification/%s", serviceName)
}

func (r *EtcdRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, cancel := range r.watchCancel {
		cancel()
	}
	return r.session.Close()
}

func NewEtcdRegistry(client *clientv3.Client) (*EtcdRegistry, error) {
	session, err := concurrency.NewSession(client)
	if err != nil {
		return nil, err
	}
	return &EtcdRegistry{
		session: session,
		client:  client,
	}, nil
}
