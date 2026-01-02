package database

import (
	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/context"
	"github.com/go-viper/mapstructure/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	DbTypeMysql = "mysql"
)

func init() {
	FactoryDialector[DbTypeMysql] = CreateDialectorMysql
}

type MysqlConfig struct {
	DSN string `mapstructure:"dsn" validate:"required"`
}

func CreateDialectorMysql(ctx *context.Context, cfg config.DbConfig) (gorm.Dialector, error) {
	dialectorCfg := MysqlConfig{}
	err := mapstructure.Decode(cfg.Config, &dialectorCfg)
	if err != nil {
		return nil, err
	}

	err = ctx.Validator.Struct(dialectorCfg)
	if err != nil {
		return nil, err
	}

	dialector := mysql.Open(dialectorCfg.DSN)

	return dialector, nil
}
