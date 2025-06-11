package isolation

import (
	"context"
	"database/sql"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CoreBizContextKey = struct{}

// DB gorm 的数据库隔离实现
type DB struct {
	coreDB    gorm.ConnPool // 核心库
	nonCoreDB gorm.ConnPool // 非核心库
	logger    *zap.Logger
}

func (db *DB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return db.selectDB(ctx).PrepareContext(ctx, query)
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.selectDB(ctx).ExecContext(ctx, query, args...)
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.selectDB(ctx).QueryContext(ctx, query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.selectDB(ctx).QueryRowContext(ctx, query, args...)
}

func (db *DB) selectDB(ctx context.Context) gorm.ConnPool {
	if db.IsCore(ctx) {
		return db.coreDB
	}
	return db.nonCoreDB
}

// IsCore 判断是否为核心业务
func (db *DB) IsCore(ctx context.Context) bool {
	val := ctx.Value(CoreBizContextKey{})
	return val != nil
}

func NewDB(coreDB, nonCoreDB gorm.ConnPool, logger *zap.Logger) *DB {
	return &DB{
		coreDB:    coreDB,
		nonCoreDB: nonCoreDB,
		logger:    logger,
	}
}

func WithCore(ctx context.Context) context.Context {
	return context.WithValue(ctx, CoreBizContextKey{}, true)
}
