package database

import (
	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/context"
	"github.com/go-viper/mapstructure/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	DbTypeSqlite = "sqlite"
)

type SqliteConfig struct {
	DSN string `mapstructure:"dsn" validate:"required"`
}

func CreateDialectorSqlite(ctx *context.Context, cfg config.DbConfig) (gorm.Dialector, error) {
	dialectorCfg := SqliteConfig{}
	err := mapstructure.Decode(cfg.Config, &dialectorCfg)
	if err != nil {
		return nil, err
	}

	err = ctx.Validator.Struct(dialectorCfg)
	if err != nil {
		return nil, err
	}

	dialector := sqlite.Open(dialectorCfg.DSN)

	return dialector, nil
}
