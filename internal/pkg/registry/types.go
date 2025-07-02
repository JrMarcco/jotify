package registry

import (
	"context"
	"io"
)

type Registry interface {
	Register(ctx context.Context, si ServiceInstance) error
	Unregister(ctx context.Context, si ServiceInstance) error
	ListService(ctx context.Context, serviceName string) ([]ServiceInstance, error)
	Subscribe(serviceName string) <-chan Event

	io.Closer
}

type ServiceInstance struct {
	Name        string
	Addr        string
	Group       string
	ReadWeight  int
	WriteWeight int
}

type EventType int

//goland:noinspection GoUnusedConst
const (
	EventTypeUnknown EventType = iota
	EventTypePut
	EventTypeDelete
)

type Event struct {
	Type            EventType
	ServiceInstance ServiceInstance
}

func (e Event) IsPut() bool {
	return e.Type == EventTypePut
}

func (e Event) IsDelete() bool {
	return e.Type == EventTypeDelete
}
