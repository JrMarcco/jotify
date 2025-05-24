package snowflake

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
)

const (
	timestampBits = 41
	hashBits      = 10
	sequenceBits  = 12

	hashShift      = sequenceBits
	timestampShift = hashShift + hashBits

	sequenceMask  = (uint64(1) << sequenceBits) - 1
	hashMask      = (uint64(1) << hashBits) - 1
	timestampMask = (uint64(1) << timestampBits) - 1

	epochMillis   = uint64(1735689600000) // milliseconds of 2025-01-01 00:00:00
	number1000    = uint64(1000)
	number1000000 = uint64(1000000)
)

// Generator 自定义雪花算法 id 生成器。
// 其中机器码的 10 位替换为由业务 id 和业务 key 计算而来的 hash 值。
// 这样通过 id 能解析出业务 id 和业务 key 从而获得分库分表信息。
type Generator struct {
	sequence uint64
	lastTime uint64    // the time of last id generated
	epoch    time.Time // the epoch time
}

func NewGenerator() *Generator {
	return &Generator{
		sequence: 0,
		lastTime: 0,
		epoch:    time.Unix(int64(epochMillis/number1000), int64((epochMillis%number1000)*number1000000)),
	}
}

// NextId 返回生成 id。
//
// id 组成
//   - 41 位时间戳，几点时间为 2025-01-01 00:00:00。
//   - 10 位 hash 值，由业务 id（bizId）和业务 key（bizKey）一起计算出的 hash 值。
//   - 12 位自增序列
func (g *Generator) NextId(bizId uint64, bizKey string) uint64 {
	timestamp := uint64(time.Now().UnixMilli()) - epochMillis
	hashVal := xxhash.Sum64String(HashKey(bizId, bizKey))

	seq := atomic.AddUint64(&g.sequence, 1) - 1

	return (timestamp&timestampMask)<<timestampShift | (hashVal&hashMask)<<hashShift | (seq & sequenceMask)
}

func HashKey(bizId uint64, bizKey string) string {
	return strconv.FormatUint(bizId, 10) + ":" + bizKey
}

func ExtractTimestamp(id uint64) time.Time {
	timestamp := (id >> timestampShift) & timestampMask
	return time.Unix(0, int64((timestamp+epochMillis)*uint64(time.Millisecond)))
}

func ExtractHash(id uint64) uint64 {
	return (id >> hashShift) & hashMask
}

func ExtractSequence(id uint64) uint64 {
	return id & sequenceMask
}
