package database

import (
	"errors"
	"testing"

	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestModels(t *testing.T) {
	t.Run("contains all expected models", func(t *testing.T) {
		expectedModels := []interface{}{
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

		assert.Equal(t, len(expectedModels), len(Models))
		for i, expected := range expectedModels {
			assert.IsType(t, expected, Models[i])
		}
	})

	t.Run("models count is 13", func(t *testing.T) {
		assert.Len(t, Models, 13)
	})
}

func TestFactoryDialector(t *testing.T) {
	t.Run("is initialized as empty map", func(t *testing.T) {
		assert.NotNil(t, FactoryDialector)
	})

	t.Run("can register and retrieve dialector factory", func(t *testing.T) {
		testType := "test-dialector"
		testFn := func(ctx *context.Context, cfg config.DbConfig) (gorm.Dialector, error) {
			return nil, nil
		}

		// Register
		FactoryDialector[testType] = testFn

		// Verify
		fn, ok := FactoryDialector[testType]
		assert.True(t, ok)
		assert.NotNil(t, fn)

		// Cleanup
		delete(FactoryDialector, testType)
	})
}

func TestCreateDB(t *testing.T) {
	// Save original state and restore after tests
	originalInstance := dbInstance
	originalFactory := make(map[string]CreateDialectorFn)
	for k, v := range FactoryDialector {
		originalFactory[k] = v
	}

	t.Cleanup(func() {
		dbInstance = originalInstance
		FactoryDialector = originalFactory
	})

	t.Run("error when db type does not exist", func(t *testing.T) {
		// Reset for this test
		dbInstance = nil
		FactoryDialector = make(map[string]CreateDialectorFn)

		ctx := context.TestContext(nil)
		ctx.Config.DB = config.DbConfig{
			Type: "nonexistent",
		}

		db, err := CreateDB(ctx)

		require.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("error when dialector factory returns error", func(t *testing.T) {
		// Reset for this test
		dbInstance = nil
		FactoryDialector = map[string]CreateDialectorFn{
			"error-type": func(ctx *context.Context, cfg config.DbConfig) (gorm.Dialector, error) {
				return nil, errors.New("factory error")
			},
		}

		ctx := context.TestContext(nil)
		ctx.Config.DB = config.DbConfig{
			Type: "error-type",
		}

		db, err := CreateDB(ctx)

		require.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "factory error")
	})

	t.Run("error when dialector returns nil", func(t *testing.T) {
		// Reset for this test
		dbInstance = nil
		FactoryDialector = map[string]CreateDialectorFn{
			"nil-type": func(ctx *context.Context, cfg config.DbConfig) (gorm.Dialector, error) {
				return nil, nil
			},
		}

		ctx := context.TestContext(nil)
		ctx.Config.DB = config.DbConfig{
			Type: "nil-type",
		}

		db, err := CreateDB(ctx)

		require.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("success with sqlite in memory", func(t *testing.T) {
		// Reset for this test
		dbInstance = nil
		FactoryDialector = map[string]CreateDialectorFn{
			DbTypeSqlite: CreateDialectorSqlite,
		}

		ctx := context.TestContext(nil)
		ctx.Config.DB = config.DbConfig{
			Type: DbTypeSqlite,
			Config: map[string]interface{}{
				"dsn": ":memory:",
			},
		}

		db, err := CreateDB(ctx)

		require.NoError(t, err)
		assert.NotNil(t, db)
	})

	t.Run("returns singleton on second call", func(t *testing.T) {
		// Don't reset - use the instance from previous test
		ctx := context.TestContext(nil)
		ctx.Config.DB = config.DbConfig{
			Type: DbTypeSqlite,
			Config: map[string]interface{}{
				"dsn": ":memory:",
			},
		}

		db1, err1 := CreateDB(ctx)
		require.NoError(t, err1)

		db2, err2 := CreateDB(ctx)
		require.NoError(t, err2)

		assert.Same(t, db1, db2)
	})
}

func TestGetGormLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    config.DbLogLevel
		expected logger.LogLevel
	}{
		{
			name:     "silent level",
			level:    config.DbLogLevelSilent,
			expected: logger.Silent,
		},
		{
			name:     "error level",
			level:    config.DbLogLevelError,
			expected: logger.Error,
		},
		{
			name:     "warn level",
			level:    config.DbLogLevelWarn,
			expected: logger.Warn,
		},
		{
			name:     "info level",
			level:    config.DbLogLevelInfo,
			expected: logger.Info,
		},
		{
			name:     "empty string defaults to silent",
			level:    "",
			expected: logger.Silent,
		},
		{
			name:     "unknown level defaults to silent",
			level:    "unknown",
			expected: logger.Silent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGormLogLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}
