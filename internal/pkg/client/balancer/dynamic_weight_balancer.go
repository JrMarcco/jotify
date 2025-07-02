package balancer

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/JrMarcco/jotify/internal/pkg/client"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ base.PickerBuilder = (*DynamicWeightBalancerBuilder)(nil)

type DynamicWeightBalancerBuilder struct{}

func (b *DynamicWeightBalancerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	nodes := make([]*dynamicServiceNode, 0, len(info.ReadySCs))

	for cc, ccInfo := range info.ReadySCs {
		weight, _ := ccInfo.Address.Attributes.Value(client.AttrWeight).(int32)
		nodes = append(nodes, &dynamicServiceNode{
			cc:              cc,
			weight:          weight,
			currentWeight:   weight,
			efficientWeight: weight,
		})
	}

	return &DynamicWeightBalancer{nodes: nodes}
}

var _ balancer.Picker = (*DynamicWeightBalancer)(nil)

type DynamicWeightBalancer struct {
	nodes []*dynamicServiceNode
}

func (b *DynamicWeightBalancer) Pick(_ balancer.PickInfo) (balancer.PickResult, error) {
	if len(b.nodes) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	var totalWeight int32
	var selectedNode *dynamicServiceNode
	for _, node := range b.nodes {
		node.mu.RLock()
		totalWeight += node.efficientWeight
		node.currentWeight += node.efficientWeight

		if selectedNode == nil || selectedNode.currentWeight < node.currentWeight {
			selectedNode = node
		}
		node.mu.RUnlock()
	}

	if selectedNode == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	selectedNode.mu.Lock()
	selectedNode.currentWeight -= totalWeight
	selectedNode.mu.Unlock()

	return balancer.PickResult{
		SubConn: selectedNode.cc,
		Done: func(info balancer.DoneInfo) {
			selectedNode.mu.Lock()
			defer selectedNode.mu.Unlock()

			if info.Err == nil {
				const twice = 2

				selectedNode.efficientWeight++
				selectedNode.efficientWeight = max(selectedNode.efficientWeight, selectedNode.weight*twice)
				return
			}

			if errors.Is(info.Err, context.DeadlineExceeded) || errors.Is(info.Err, io.EOF) {
				selectedNode.efficientWeight = 1
				return
			}

			res, _ := status.FromError(info.Err)
			switch res.Code() {
			case codes.Unavailable:
				selectedNode.efficientWeight = 1
				return
			default:
				if selectedNode.efficientWeight > 1 {
					selectedNode.efficientWeight--
				}
			}
		},
	}, nil
}

type dynamicServiceNode struct {
	mu sync.RWMutex

	cc              balancer.SubConn
	weight          int32
	currentWeight   int32
	efficientWeight int32
}
