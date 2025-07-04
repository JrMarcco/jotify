package balancer

import (
	"sync/atomic"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

var _ base.PickerBuilder = (*RoundRobinBalancerBuilder)(nil)

type RoundRobinBalancerBuilder struct{}

func (b *RoundRobinBalancerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	ccs := make([]balancer.SubConn, 0, len(info.ReadySCs))

	for cc := range info.ReadySCs {
		ccs = append(ccs, cc)
	}

	return &RoundRobinBalancer{
		ccs:    ccs,
		index:  -1,
		length: uint64(len(ccs)),
	}
}

var _ balancer.Picker = (*RoundRobinBalancer)(nil)

type RoundRobinBalancer struct {
	ccs []balancer.SubConn

	index  uint64
	length uint64
}

func (p *RoundRobinBalancer) Pick(_ balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.ccs) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	index := atomic.AddUint64(&p.index, 1)
	cc := p.ccs[index%p.length]
	return balancer.PickResult{
		SubConn: cc,
		Done:    func(_ balancer.DoneInfo) {},
	}, nil
}
