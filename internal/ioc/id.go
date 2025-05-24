package ioc

import (
	"github.com/JrMarcco/jotify/internal/pkg/snowflake"
	"go.uber.org/fx"
)

var IdFxOpt = fx.Provide(
	InitIdGenerator,
)

func InitIdGenerator() *snowflake.Generator {
	return snowflake.NewGenerator()
}
