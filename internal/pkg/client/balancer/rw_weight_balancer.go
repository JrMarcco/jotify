package balancer

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/JrMarcco/easy-kit/slice"
	"github.com/JrMarcco/jotify/internal/pkg/client"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

var _ base.PickerBuilder = (*RwWeightBalancerBuilder)(nil)

type RwWeightBalancerBuilder struct {
	mu        sync.RWMutex
	nodeCache map[string]*rwWeightServiceNode
}

func (b *RwWeightBalancerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	nodes := make([]*rwWeightServiceNode, 0, len(info.ReadySCs))
	ccMap := make(map[string]struct{})

	b.mu.Lock()
	defer b.mu.Unlock()

	for cc, ccInfo := range info.ReadySCs {
		readWeight, ok := ccInfo.Address.Attributes.Value(client.AttrReadWeight).(int32)
		if !ok {
			continue
		}
		writeWeight, ok := ccInfo.Address.Attributes.Value(client.AttrWriteWeight).(int32)
		if !ok {
			continue
		}
		groupName, ok := ccInfo.Address.Attributes.Value(client.AttrGroup).(string)
		if !ok {
			continue
		}
		nodeName, ok := ccInfo.Address.Attributes.Value(client.AttrNode).(string)
		if !ok {
			continue
		}
		ccMap[nodeName] = struct{}{}

		if cacheNode, ok := b.nodeCache[nodeName]; ok {
			// 当前节点存在缓存中，更新连接信息、组信息
			cacheNode.mu.Lock()
			cacheNode.group = groupName
			cacheNode.mu.Unlock()

			if cacheNode.readWeight != readWeight || cacheNode.writeWeight != writeWeight {
				// 权重发生变化，更新权重
				cacheNode = newRwWeightServiceNode(cc, readWeight, writeWeight, groupName)
				b.nodeCache[nodeName] = cacheNode
			}
			nodes = append(nodes, cacheNode)
		} else {
			newNode := newRwWeightServiceNode(cc, readWeight, writeWeight, groupName)
			b.nodeCache[nodeName] = newNode
			nodes = append(nodes, newNode)
		}
	}

	// 从缓存中清除已经不存在的节点
	for key := range b.nodeCache {
		if _, ok := ccMap[key]; !ok {
			delete(b.nodeCache, key)
		}
	}

	return &RwReadWeightBalancer{nodes: nodes}
}

func NewRwWeightBalancerBuilder() *RwWeightBalancerBuilder {
	return &RwWeightBalancerBuilder{
		nodeCache: make(map[string]*rwWeightServiceNode),
	}
}

func WithGroup(ctx context.Context, group string) context.Context {
	return context.WithValue(ctx, client.ContextKeyGroup{}, group)
}

var _ balancer.Picker = (*RwReadWeightBalancer)(nil)

type RwReadWeightBalancer struct {
	nodes []*rwWeightServiceNode
}

func (p *RwReadWeightBalancer) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.nodes) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	// 获取候选节点
	ctx := info.Ctx
	candidateNodes := slice.FilterMap(p.nodes, func(_ int, src *rwWeightServiceNode) (*rwWeightServiceNode, bool) {
		src.mu.RLock()
		nodeGroup := src.group
		src.mu.RUnlock()
		return src, p.getGroup(ctx) == nodeGroup
	})

	if len(candidateNodes) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	var totalWeight int32
	var selectedNode *rwWeightServiceNode

	isWriteReq := p.isWriteReq(ctx)

	// 权重计算
	for _, node := range candidateNodes {
		node.mu.Lock()
		if isWriteReq {
			totalWeight += node.efficientWriteWeight
			node.currentWriteWeight += node.efficientWriteWeight
			if selectedNode == nil || selectedNode.currentWriteWeight < node.currentWriteWeight {
				selectedNode = node
			}
			node.mu.Unlock()
			continue
		}

		totalWeight += node.efficientReadWeight
		node.currentReadWeight += node.efficientReadWeight
		if selectedNode == nil || selectedNode.currentReadWeight < node.currentReadWeight {
			selectedNode = node
		}
		node.mu.Unlock()
	}

	if selectedNode == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	selectedNode.mu.Lock()
	if isWriteReq {
		selectedNode.currentWriteWeight -= totalWeight
	} else {
		selectedNode.currentReadWeight -= totalWeight
	}
	selectedNode.mu.Unlock()

	return balancer.PickResult{
		SubConn: selectedNode.cc,
		Done: func(info balancer.DoneInfo) {
			selectedNode.mu.Lock()
			defer selectedNode.mu.Unlock()

			isDecrementErr := info.Err != nil && (errors.Is(info.Err, context.DeadlineExceeded) || errors.Is(info.Err, io.EOF))
			const twice = 2
			if isWriteReq {
				if info.Err == nil {
					selectedNode.efficientWriteWeight++
					selectedNode.currentWriteWeight = max(selectedNode.efficientWriteWeight, selectedNode.writeWeight*twice)
					return
				}
				if isDecrementErr && selectedNode.efficientWriteWeight > 1 {
					selectedNode.efficientWriteWeight--
				}
				return
			}

			if info.Err == nil {
				selectedNode.efficientReadWeight++
				selectedNode.currentReadWeight = max(selectedNode.efficientReadWeight, selectedNode.readWeight*twice)
				return
			}

			if isDecrementErr && selectedNode.efficientReadWeight > 1 {
				selectedNode.efficientReadWeight--
			}
		},
	}, nil
}

func (p *RwReadWeightBalancer) getGroup(ctx context.Context) string {
	val := ctx.Value(client.ContextKeyGroup{})
	if val == nil {
		return ""
	}

	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

func (p *RwReadWeightBalancer) isWriteReq(ctx context.Context) bool {
	val := ctx.Value(client.ContextKeyReqType{})
	if val == nil {
		return false
	}

	if intVal, ok := val.(int); ok {
		return intVal == 1
	}
	return false
}

type rwWeightServiceNode struct {
	mu sync.RWMutex

	cc                  balancer.SubConn
	readWeight          int32
	currentReadWeight   int32
	efficientReadWeight int32

	writeWeight          int32
	currentWriteWeight   int32
	efficientWriteWeight int32

	group string
}

func newRwWeightServiceNode(cc balancer.SubConn, readWeight int32, writeWeight int32, group string) *rwWeightServiceNode {
	return &rwWeightServiceNode{
		cc:                   cc,
		readWeight:           readWeight,
		currentReadWeight:    readWeight,
		efficientReadWeight:  readWeight,
		writeWeight:          writeWeight,
		currentWriteWeight:   writeWeight,
		efficientWriteWeight: writeWeight,
		group:                group,
	}
}
