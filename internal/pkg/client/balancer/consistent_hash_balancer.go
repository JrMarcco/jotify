package balancer

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

var _ base.PickerBuilder = (*ConsistentHashBalancerBuilder)(nil)

type ConsistentHashBalancerBuilder struct{}

func (b *ConsistentHashBalancerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return &ConsistentHashBalancer{}
	}

	hashRing := NewHashRing()

	// 为每个连接添加虚拟节点
	for cc, ccInfo := range info.ReadySCs {
		// 使用连接的地址作为节点标识
		nodeID := ccInfo.Address.Addr
		hashRing.AddNode(nodeID, cc)
	}

	return &ConsistentHashBalancer{
		hashRing: hashRing,
	}
}

var _ balancer.Picker = (*ConsistentHashBalancer)(nil)

type ConsistentHashBalancer struct {
	hashRing *HashRing
}

func (p *ConsistentHashBalancer) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if p.hashRing == nil || p.hashRing.IsEmpty() {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	// 使用 FullMethodName 作为哈希键
	// 你也可以根据需要使用其他信息，比如客户端 IP、用户 ID 等
	key := info.FullMethodName
	if key == "" {
		key = "default"
	}

	conn := p.hashRing.GetNode(key)
	if conn == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	subConn, ok := conn.(balancer.SubConn)
	if !ok {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	return balancer.PickResult{
		SubConn: subConn,
		Done:    func(_ balancer.DoneInfo) {},
	}, nil
}

// HashRing 一致性哈希环
type HashRing struct {
	mu           sync.RWMutex
	virtualNodes int            // 每个物理节点的虚拟节点数量
	keys         []uint32       // 哈希环上的键值，已排序
	ring         map[uint32]any // 哈希环，键是哈希值，值是连接
	nodes        map[string]any // 节点映射，键是节点ID，值是连接
}

// NewHashRing 创建新的哈希环
func NewHashRing() *HashRing {
	return &HashRing{
		virtualNodes: 150, // 每个物理节点150个虚拟节点
		ring:         make(map[uint32]any),
		nodes:        make(map[string]any),
	}
}

// AddNode 添加节点到哈希环
func (h *HashRing) AddNode(nodeID string, conn any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nodes[nodeID] = conn

	// 为这个节点创建虚拟节点
	for i := 0; i < h.virtualNodes; i++ {
		virtualKey := fmt.Sprintf("%s#%d", nodeID, i)
		hash := h.hash(virtualKey)
		h.ring[hash] = conn
		h.keys = append(h.keys, hash)
	}

	// 排序哈希环的键
	sort.Slice(h.keys, func(i, j int) bool {
		return h.keys[i] < h.keys[j]
	})
}

// RemoveNode 从哈希环中移除节点
func (h *HashRing) RemoveNode(nodeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.nodes[nodeID]; !exists {
		return
	}

	delete(h.nodes, nodeID)

	// 移除虚拟节点
	for i := 0; i < h.virtualNodes; i++ {
		virtualKey := fmt.Sprintf("%s#%d", nodeID, i)
		hash := h.hash(virtualKey)
		delete(h.ring, hash)
	}

	// 重建键列表
	h.keys = h.keys[:0]
	for hash := range h.ring {
		h.keys = append(h.keys, hash)
	}
	sort.Slice(h.keys, func(i, j int) bool {
		return h.keys[i] < h.keys[j]
	})
}

// GetNode 根据键获取对应的节点
func (h *HashRing) GetNode(key string) any {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.keys) == 0 {
		return nil
	}

	hash := h.hash(key)

	// 在哈希环上查找第一个大于等于hash的节点
	idx := sort.Search(len(h.keys), func(i int) bool {
		return h.keys[i] >= hash
	})

	// 如果没有找到，则使用第一个节点（环形结构）
	if idx == len(h.keys) {
		idx = 0
	}

	return h.ring[h.keys[idx]]
}

// IsEmpty 检查哈希环是否为空
func (h *HashRing) IsEmpty() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.keys) == 0
}

// hash 计算字符串的哈希值
func (h *HashRing) hash(key string) uint32 {
	md5Hash := md5.Sum([]byte(key))
	return binary.BigEndian.Uint32(md5Hash[:4])
}
