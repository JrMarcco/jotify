package bitring

import (
	"sync"
)

const (
	bitsPerWord = 64              // uint64 位数
	bitsMask    = bitsPerWord - 1 // 位操作掩码: 0x3f
	bitsShift   = 6               // 位计算偏移量: log2(64)

	defaultWindowSize     = 128 // 默认窗口大小
	defaultMinConsecutive = 3   // 默认最小连续时间数

)

// BitRing 一个使用比特环记录事件的滑动窗口
type BitRing struct {
	mu sync.RWMutex

	words []uint64 // 存放事件状态的比特环

	windowSize int // 窗口大小
	writePos   int // 下一个事件状态写入位置

	isFull               bool
	eventCount           int
	consecutiveThreshold int     // 最小连续时间数阈值
	eventRateThreshold   float64 // 事件发生率阈值
}

func (br *BitRing) Add(eventHappened bool) {
	br.mu.Lock()
	defer br.mu.Unlock()

	oldBit := br.bitAt(br.writePos)
	if br.isFull && oldBit {
		// 当前比特环已满且原来位置标记的是事件发生
		// 事件总数 - 1（本次写入会覆盖，如果写入的事件为发生再加回计数）
		br.eventCount--
	}
	br.setBit(br.writePos, eventHappened)
	if eventHappened {
		br.eventCount++
	}

	br.writePos++
	if br.writePos >= br.windowSize {
		br.writePos = 0
		br.isFull = true
	}
}

func (br *BitRing) bitAt(index int) bool {
	pos := index >> bitsShift
	offset := uint(index & bitsMask)
	return (br.words[pos]>>offset)&1 == 1
}

func (br *BitRing) setBit(index int, val bool) {
	pos := index >> bitsShift
	offset := uint(index & bitsMask)

	if val {
		// 设置为 1
		br.words[pos] |= 1 << offset
		return
	}
	// 把指定位置（pos）设置为 0
	// &^ 是 Go 特有的位运算操作，用于在操作符右侧为 1 时将操作符左侧操作数位清零。
	br.words[pos] &^= 1 << offset
}

// ThresholdTriggering 判断阈值是否出触发
func (br *BitRing) ThresholdTriggering() bool {
	br.mu.RLock()
	defer br.mu.RUnlock()

	currSize := br.currWindowSize()
	if currSize == 0 {
		return false
	}

	if currSize >= br.consecutiveThreshold {
		all := true
		for i := 1; i <= br.consecutiveThreshold; i++ {
			pos := (br.writePos - i + br.windowSize) % br.windowSize
			if br.bitAt(pos) {
				continue
			}
			all = false
			break
		}
		if all {
			return true
		}
	}

	if float64(br.eventCount)/float64(currSize) > br.eventRateThreshold {
		return true
	}
	return false
}

func (br *BitRing) currWindowSize() int {
	if br.isFull {
		return br.windowSize
	}
	return br.writePos
}

func NewBitRing(windowSize int, consecutiveThreshold int, eventRateThreshold float64) *BitRing {
	if windowSize <= 0 {
		windowSize = defaultWindowSize
	}

	if consecutiveThreshold <= 0 {
		consecutiveThreshold = defaultMinConsecutive
	}

	if consecutiveThreshold > windowSize {
		consecutiveThreshold = windowSize
	}

	if eventRateThreshold > 1 {
		eventRateThreshold = 1
	}

	return &BitRing{
		words:                make([]uint64, (windowSize+bitsMask)/bitsPerWord),
		windowSize:           windowSize,
		consecutiveThreshold: consecutiveThreshold,
		eventRateThreshold:   eventRateThreshold,
	}
}
