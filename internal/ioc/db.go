package ioc

import (
	"sync"

	"github.com/JrMarcco/easy-kit/xsync"
	"github.com/JrMarcco/jotify/internal/pkg/sharding"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DBFxOpt = fx.Provide(
	InitBaseDB,
	InitShardingDB,
	fx.Annotate(
		InitNotifShardingSharding,
		fx.As(new(sharding.Strategy)),
	),
)

var (
	mu   sync.Mutex
	once sync.Once
)

func InitBaseDB() *gorm.DB {
	type dbConfig struct {
		DSN string `mapstructure:"dsn"`
	}
	cfg := &dbConfig{}
	if err := viper.UnmarshalKey("db.base", cfg); err != nil {
		panic(err)
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}

func InitShardingDB() *xsync.Map[string, *gorm.DB] {
	type dbConfig struct {
		DSN string `mapstructure:"dsn"`
	}

	type allDBConfig map[string]dbConfig
	dbConfigs := make(allDBConfig)

	if err := viper.UnmarshalKey("db.sharding", &dbConfigs); err != nil {
		panic(err)
	}

	mu.Lock()
	defer mu.Unlock()

	var dbs xsync.Map[string, *gorm.DB]
	once.Do(func() {
		for key, cfg := range dbConfigs {
			db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
			if err != nil {
				panic(err)
			}
			dbs.Store(key, db)
		}
	})
	return &dbs
}

func InitNotifShardingSharding() sharding.HashStrategy {
	return sharding.NewHashStrategy(
		"jotify", "notification", 2, 4,
	)
}
