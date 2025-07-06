package balancer

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

var _ base.PickerBuilder = (*HashBalancerBuilder)(nil)

type HashBalancerBuilder struct{}

func (b *HashBalancerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	//TODO implement me
	panic("implement me")
}

var _ balancer.Picker = (*HashBalancer)(nil)

type HashBalancer struct {
}

func (p *HashBalancer) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	//TODO implement me
	panic("implement me")
}
