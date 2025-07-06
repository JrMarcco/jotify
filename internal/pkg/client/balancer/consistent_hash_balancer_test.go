package balancer

import (
	"testing"
)

// mockConn 简单的模拟连接对象
type mockConn struct {
	id string
}

func TestHashRing_Basic(t *testing.T) {
	ring := NewHashRing()

	// 测试空环
	if !ring.IsEmpty() {
		t.Fatal("Expected ring to be empty")
	}

	node := ring.GetNode("test-key")
	if node != nil {
		t.Fatal("Expected nil node for empty ring")
	}
}

func TestHashRing_SingleNode(t *testing.T) {
	ring := NewHashRing()

	// 模拟一个连接对象
	conn := &mockConn{id: "server-1"}

	ring.AddNode("127.0.0.1:8001", conn)

	if ring.IsEmpty() {
		t.Fatal("Expected ring to be non-empty")
	}

	// 测试相同的key应该返回相同的节点
	node1 := ring.GetNode("test-key-1")
	node2 := ring.GetNode("test-key-1")

	if node1 != node2 {
		t.Fatal("Expected same node for same key")
	}

	if node1 != conn {
		t.Fatal("Expected correct node")
	}
}

func TestHashRing_MultipleNodes(t *testing.T) {
	ring := NewHashRing()

	conn1 := &mockConn{id: "server-1"}
	conn2 := &mockConn{id: "server-2"}
	conn3 := &mockConn{id: "server-3"}

	ring.AddNode("127.0.0.1:8001", conn1)
	ring.AddNode("127.0.0.1:8002", conn2)
	ring.AddNode("127.0.0.1:8003", conn3)

	// 测试负载分布
	nodeUsage := make(map[interface{}]int)
	for i := 0; i < 1000; i++ {
		key := generateKey(i)
		node := ring.GetNode(key)
		nodeUsage[node]++
	}

	// 验证所有节点都被使用
	if len(nodeUsage) != 3 {
		t.Fatalf("Expected 3 nodes to be used, got %d", len(nodeUsage))
	}

	// 验证负载分布相对均匀
	for node, count := range nodeUsage {
		if count < 200 || count > 500 {
			t.Fatalf("Node %v has uneven distribution: %d", node, count)
		}
	}
}

func TestHashRing_Consistency(t *testing.T) {
	ring := NewHashRing()

	conn1 := &mockConn{id: "server-1"}
	conn2 := &mockConn{id: "server-2"}

	// 添加第一个节点
	ring.AddNode("127.0.0.1:8001", conn1)

	// 记录1000个键的分布
	distribution1 := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		key := generateKey(i)
		distribution1[key] = ring.GetNode(key)
	}

	// 添加第二个节点
	ring.AddNode("127.0.0.1:8002", conn2)

	// 记录添加节点后的分布
	distribution2 := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		key := generateKey(i)
		distribution2[key] = ring.GetNode(key)
	}

	// 计算变化的键数量
	changedKeys := 0
	for key := range distribution1 {
		if distribution1[key] != distribution2[key] {
			changedKeys++
		}
	}

	// 一致性哈希应该保证大部分键不变
	changeRate := float64(changedKeys) / 1000.0
	if changeRate > 0.6 { // 允许60%的变化率
		t.Fatalf("Too many keys changed: %f", changeRate)
	}
}

func TestHashRing_RemoveNode(t *testing.T) {
	ring := NewHashRing()

	conn1 := &mockConn{id: "server-1"}
	conn2 := &mockConn{id: "server-2"}

	// 添加两个节点
	ring.AddNode("127.0.0.1:8001", conn1)
	ring.AddNode("127.0.0.1:8002", conn2)

	// 移除第一个节点
	ring.RemoveNode("127.0.0.1:8001")

	// 验证所有请求都路由到第二个节点
	for i := 0; i < 100; i++ {
		key := generateKey(i)
		node := ring.GetNode(key)
		if node != conn2 {
			t.Fatalf("Expected all requests to go to conn2, got %v", node)
		}
	}

	// 移除最后一个节点
	ring.RemoveNode("127.0.0.1:8002")

	if !ring.IsEmpty() {
		t.Fatal("Expected ring to be empty after removing all nodes")
	}
}

func TestHashRing_SameKeyConsistency(t *testing.T) {
	ring := NewHashRing()

	conn1 := &mockConn{id: "server-1"}
	conn2 := &mockConn{id: "server-2"}

	ring.AddNode("127.0.0.1:8001", conn1)
	ring.AddNode("127.0.0.1:8002", conn2)

	// 测试相同的key多次请求应该返回相同的节点
	testKey := "consistent-key"
	firstNode := ring.GetNode(testKey)

	for i := 0; i < 100; i++ {
		node := ring.GetNode(testKey)
		if node != firstNode {
			t.Fatalf("Expected consistent node for same key, got different nodes")
		}
	}
}

func TestHash_Function(t *testing.T) {
	ring := NewHashRing()

	// 测试不同的键产生不同的哈希值
	hash1 := ring.hash("key1")
	hash2 := ring.hash("key2")
	hash3 := ring.hash("key1") // 相同的键

	if hash1 == hash2 {
		t.Fatal("Expected different hash values for different keys")
	}

	if hash1 != hash3 {
		t.Fatal("Expected same hash value for same key")
	}
}

// generateKey 生成测试用的键
func generateKey(i int) string {
	return "key-" + string(rune('0'+i%10)) + string(rune('a'+i%26))
}
