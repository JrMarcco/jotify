package balancer

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/JrMarcco/jotify/internal/pkg/client"
	"github.com/JrMarcco/jotify/internal/pkg/snowflake"
	"github.com/cespare/xxhash/v2"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

var _ base.PickerBuilder = (*CHBalancerBuilder)(nil)

type CHBalancerBuilder struct {
	virtualNodeCnt int // 虚拟节点数量
}

func (b *CHBalancerBuilder) VirtualNodeCnt(cnt int) {
	b.virtualNodeCnt = cnt
}

func (b *CHBalancerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	if b.virtualNodeCnt <= 0 {
		return base.NewErrPicker(fmt.Errorf("[jotify] virtual node count must be greater than 0"))
	}

	ccs := make(map[string]balancer.SubConn, len(info.ReadySCs))
	ring := make([]uint32, 0, b.virtualNodeCnt*len(info.ReadySCs))
	nodes := make([]string, 0, b.virtualNodeCnt*len(info.ReadySCs))
	p := &CHBalancer{
		ccs:            ccs,
		ring:           ring,
		nodes:          nodes,
		virtualNodeCnt: b.virtualNodeCnt,
	}

	for cc, ccInfo := range info.ReadySCs {
		p.addNode(cc, ccInfo.Address.Addr)
	}

	return p
}

func NewCHBalancerBuilder() *CHBalancerBuilder {
	return &CHBalancerBuilder{
		virtualNodeCnt: 100,
	}
}

var _ balancer.Picker = (*CHBalancer)(nil)

type CHBalancer struct {
	mu sync.RWMutex

	ccs            map[string]balancer.SubConn // 虚拟节点地址 -> SubConn
	ring           []uint32                    // 哈希环，存储虚拟节点的哈希值
	nodes          []string                    // 与哈希环对应的虚拟节点地址
	virtualNodeCnt int                         // 虚拟节点数量
}

func (p *CHBalancer) addNode(cc balancer.SubConn, addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ccs[addr] = cc

	// 为每个物理节点创建多个虚拟节点
	// 虚拟节点在一致性哈希中起到**关键的负载均衡作用
	//
	// 假设有 3 个物理节点：
	// 哈希后可能分布在环上的位置：
	// 		s1: hash=100
	// 		s2: hash=200
	// 		s3: hash=300
	// 在为每个物理节点创建 100 个虚拟节点后：
	// 使用虚拟节点的哈希环：
	// 0 - s1 - s2 - s3 - s1 - s2 - s3 - s1 - s2 - s3 - ... - s1 - s2 - 1000
	// 负载分布：s1(33%), s2(33%), s3(34%)
	for i := 0; i < p.virtualNodeCnt; i++ {
		hash := p.hash(fmt.Sprintf("%s#%d", addr, i))

		p.ring = append(p.ring, hash)
		p.nodes = append(p.nodes, addr)
	}

	p.sortRing()
}

func (p *CHBalancer) sortRing() {
	// 索引切片，用于排序
	indices := make([]int, len(p.ring))
	for i := range indices {
		indices[i] = i
	}

	// 根据哈希值排序索引
	sort.Slice(indices, func(i, j int) bool {
		return p.ring[indices[i]] < p.ring[indices[j]]
	})

	sortedRing := make([]uint32, len(p.ring))
	sortedNodes := make([]string, len(p.nodes))
	for i, index := range indices {
		sortedRing[i] = p.ring[index]
		sortedNodes[i] = p.nodes[index]
	}

	p.ring = sortedRing
	p.nodes = sortedNodes
}

func (p *CHBalancer) removeNode(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.ccs, addr)

	newRing := make([]uint32, 0, len(p.ring)-p.virtualNodeCnt)
	newNodes := make([]string, 0, len(p.nodes)-p.virtualNodeCnt)
	for i, node := range p.nodes {
		if node != addr {
			newRing = append(newRing, p.ring[i])
			newNodes = append(newNodes, node)
		}
	}

	p.ring = newRing
	p.nodes = newNodes
}

func (p *CHBalancer) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.ccs) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	hashKey, err := p.hashFromContext(info.Ctx)
	if err != nil {
		return balancer.PickResult{}, err
	}

	nodeAddr := p.getNodeAddr(hashKey)
	if nodeAddr == "" {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	cc, ok := p.ccs[nodeAddr]
	if !ok {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	return balancer.PickResult{
		SubConn: cc,
		Done:    func(_ balancer.DoneInfo) {},
	}, nil
}

func (p *CHBalancer) getNodeAddr(hash uint32) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.ring) == 0 {
		return ""
	}

	// 查找第一个大于等于当前 hash 值的节点
	index := sort.Search(len(p.ring), func(i int) bool {
		return p.ring[i] >= hash
	})

	// 没找到则取第一个节点（环结构）
	if index >= len(p.ring) {
		index = 0
	}
	return p.nodes[index]
}

func (p *CHBalancer) hashFromContext(ctx context.Context) (uint32, error) {
	// 获取 bizId
	bizId, ok := ctx.Value(client.ContextKeyBizId{}).(uint64)
	if !ok {
		return 0, fmt.Errorf("[jotify] bizId not found in context")
	}
	bizKey, ok := ctx.Value(client.ContextKeyBizKey{}).(string)
	if !ok {
		return 0, fmt.Errorf("[jotify] bizKey not found in context")
	}

	return p.hash(snowflake.HashKey(bizId, bizKey)), nil
}

// hash 这里取低 32 位就够了
func (p *CHBalancer) hash(src string) uint32 {
	hashVal := xxhash.Sum64String(src)
	return uint32(hashVal)
}
