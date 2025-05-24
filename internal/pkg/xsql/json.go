package xsql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JsonColumn 自定义数据库 json 字段类型。
//
// 如果 T 是指针且为 nil 则 Valid 必须为 false。
type JsonColumn[T any] struct {
	Val   T
	Valid bool
}

//goland:noinspection GoMixedReceiverTypes
func (j JsonColumn[T]) Value() (driver.Value, error) {
	if !j.Valid {
		return nil, nil
	}

	res, err := json.Marshal(j.Val)
	return res, err
}

//goland:noinspection GoMixedReceiverTypes
func (j *JsonColumn[T]) Scan(src interface{}) error {
	var bs []byte
	switch val := src.(type) {
	case nil:
		return nil
	case []byte:
		bs = val
	case string:
		bs = []byte(val)
	default:
		return fmt.Errorf("[jotify] unsupported type: %T", src)
	}

	if err := json.Unmarshal(bs, &j.Val); err != nil {
		return err
	}
	j.Valid = true
	return nil
}
