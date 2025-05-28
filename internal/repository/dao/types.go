package dao

import (
	"fmt"
	"strings"
)

// IsIdDuplicateErr 判断是否主键冲突。
//
// 防止雪花生成冲突 id。
// 当 id 冲突时则重新生成 id 插入。
func IsIdDuplicateErr(ids []uint64, err error) bool {
	for _, id := range ids {
		if strings.Contains(err.Error(), fmt.Sprintf("%d", id)) {
			return true
		}
	}
	return false
}
