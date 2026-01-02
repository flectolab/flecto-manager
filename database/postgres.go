package database

import (
	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/context"
	"github.com/go-viper/mapstructure/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	DbTypePostgresSql = "postgresql"
)

func init() {
	FactoryDialector[DbTypePostgresSql] = CreateDialectorPostgresSql
}

type PostgresSqlConfig struct {
	DSN string `mapstructure:"dsn" validate:"required"`
}

func CreateDialectorPostgresSql(ctx *context.Context, cfg config.DbConfig) (gorm.Dialector, error) {
	dialectorCfg := PostgresSqlConfig{}
	err := mapstructure.Decode(cfg.Config, &dialectorCfg)
	if err != nil {
		return nil, err
	}

	err = ctx.Validator.Struct(dialectorCfg)
	if err != nil {
		return nil, err
	}

	dialector := postgres.Open(dialectorCfg.DSN)

	return dialector, nil
}
