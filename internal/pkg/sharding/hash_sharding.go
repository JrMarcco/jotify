package sharding

import (
	"fmt"
	"strings"

	"github.com/JrMarcco/jotify/internal/pkg/snowflake"
	"github.com/cespare/xxhash/v2"
)

var _ Strategy = (*HashStrategy)(nil)

// HashStrategy hash 实现的分库分表策略。
type HashStrategy struct {
	dbPrefix    string
	tablePrefix string

	dbSharding    uint64
	tableSharding uint64
}

// Shard 根据 bizId 和 bizKey 进行分库分表
func (h HashStrategy) Shard(bizId uint64, bizKey string) Dst {
	hashVal := xxhash.Sum64String(snowflake.HashKey(bizId, bizKey))
	dbSuffix := hashVal % h.dbSharding
	tableSuffix := (hashVal / h.dbSharding) % h.tableSharding
	return Dst{
		DBSuffix:    dbSuffix,
		TableSuffix: tableSuffix,
		DB:          fmt.Sprintf("%s_%d", h.dbPrefix, dbSuffix),
		Table:       fmt.Sprintf("%s_%d", h.tablePrefix, tableSuffix),
	}
}

// ShardWithId 解析 id 获得分库分表信息
func (h HashStrategy) ShardWithId(id uint64) Dst {
	hashVal := snowflake.ExtractHash(id)
	dbSuffix := hashVal % h.dbSharding
	tableSuffix := (hashVal / h.dbSharding) % h.tableSharding
	return Dst{
		DBSuffix:    dbSuffix,
		TableSuffix: tableSuffix,
		DB:          fmt.Sprintf("%s_%d", h.dbPrefix, dbSuffix),
		Table:       fmt.Sprintf("%s_%d", h.tablePrefix, tableSuffix),
	}
}

// BroadCast 广播
func (h HashStrategy) BroadCast() []Dst {
	res := make([]Dst, 0, h.dbSharding*h.tableSharding)
	for i := uint64(0); i < h.dbSharding; i++ {
		for j := uint64(0); j < h.tableSharding; j++ {
			res = append(res, Dst{
				DBSuffix:    i,
				TableSuffix: j,
				DB:          fmt.Sprintf("%s_%d", h.dbPrefix, i),
				Table:       fmt.Sprintf("%s_%d", h.tablePrefix, j),
			})
		}
	}

	return res
}

func (h HashStrategy) TablePrefix() string {
	return h.tablePrefix
}

func (h HashStrategy) ExtractSuffixAndFormat(tableName string) string {
	splits := strings.Split(tableName, "_")
	suffix := splits[len(splits)-1]
	return fmt.Sprintf("%s_%s", h.tablePrefix, suffix)
}

func NewHashStrategy(dbPrefix, tablePrefix string, dbSharding, tableSharding uint64) HashStrategy {
	return HashStrategy{
		dbPrefix:      dbPrefix,
		tablePrefix:   tablePrefix,
		dbSharding:    dbSharding,
		tableSharding: tableSharding,
	}
}
