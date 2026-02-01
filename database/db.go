package database

import (
	"fmt"
	"sync"

	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbInstance *gorm.DB
	mutex      = &sync.Mutex{}
	Models     = []interface{}{
		model.Namespace{},
		model.Project{},
		model.User{},
		model.Redirect{},
		model.RedirectDraft{},
		model.Page{},
		model.PageDraft{},
		model.ResourcePermission{},
		model.AdminPermission{},
		model.Role{},
		model.UserRole{},
		model.Agent{},
		model.Token{},
	}
)

var FactoryDialector = map[string]CreateDialectorFn{}

type CreateDialectorFn func(ctx *context.Context, cfg config.DbConfig) (gorm.Dialector, error)

func CreateDB(ctx *context.Context) (*gorm.DB, error) {
	if dbInstance == nil {
		mutex.Lock()
		defer mutex.Unlock()
		dbConfig := ctx.Config.DB
		dbCfg := &gorm.Config{
			Logger: logger.NewSlogLogger(ctx.Logger, logger.Config{LogLevel: getGormLogLevel(dbConfig.LogLevel), Colorful: true}),
		}
		var err error
		var dialector gorm.Dialector
		if fn, ok := FactoryDialector[dbConfig.Type]; ok {
			dialector, err = fn(ctx, dbConfig)
			if err != nil {
				return nil, err
			}
		}

		if dialector == nil {
			return nil, fmt.Errorf("config db type '%s' does not exist", dbConfig.Type)
		}

		db, errDbOpen := gorm.Open(dialector, dbCfg)
		if errDbOpen != nil {
			return nil, fmt.Errorf("DB: failed to create database connexion: %v", errDbOpen)
		}

		dbInstance = db
	}
	return dbInstance, nil
}

// getGormLogLevel converts DbLogLevel to gorm logger.LogLevel
func getGormLogLevel(level config.DbLogLevel) logger.LogLevel {
	switch level {
	case config.DbLogLevelError:
		return logger.Error
	case config.DbLogLevelWarn:
		return logger.Warn
	case config.DbLogLevelInfo:
		return logger.Info
	default:
		return logger.Silent
	}
}
