package client

import (
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

var _ base.PickerBuilder = (*WeightBalancerBuilder)(nil)

type WeightBalancerBuilder struct{}

func (b *WeightBalancerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	nodes := make([]*weightServiceNode, 0, len(info.ReadySCs))
	totalWeight := int32(0)

	for cc, ccInfo := range info.ReadySCs {
		weight, _ := ccInfo.Address.Attributes.Value(attrWeight).(int32)
		totalWeight += weight

		nodes = append(nodes, &weightServiceNode{
			cc:            cc,
			weight:        weight,
			currentWeight: weight,
		})
	}

	return &WeightBalancer{
		nodes:       nodes,
		totalWeight: totalWeight,
	}
}

var _ balancer.Picker = (*WeightBalancer)(nil)

type WeightBalancer struct {
	nodes       []*weightServiceNode
	totalWeight int32
}

func (b *WeightBalancer) Pick(_ balancer.PickInfo) (balancer.PickResult, error) {
	if len(b.nodes) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	var selectedNode *weightServiceNode
	for _, node := range b.nodes {
		node.mu.Lock()
		node.currentWeight = node.currentWeight + node.weight

		if selectedNode == nil || selectedNode.currentWeight < node.currentWeight {
			selectedNode = node
		}

		node.mu.Unlock()
	}

	if selectedNode == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	selectedNode.mu.Lock()
	selectedNode.currentWeight -= selectedNode.weight
	selectedNode.mu.Unlock()

	return balancer.PickResult{
		SubConn: selectedNode.cc,
		Done:    func(_ balancer.DoneInfo) {},
	}, nil
}

type weightServiceNode struct {
	mu sync.RWMutex

	cc            balancer.SubConn
	weight        int32
	currentWeight int32
}
