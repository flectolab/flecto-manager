package database

import (
	"testing"

	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDbTypeSqliteConstant(t *testing.T) {
	assert.Equal(t, "sqlite", DbTypeSqlite)
}

func TestCreateDialectorSqlite(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.TestContext(nil)
		cfg := config.DbConfig{
			Type: DbTypeSqlite,
			Config: map[string]interface{}{
				"dsn": ":memory:",
			},
		}

		dialector, err := CreateDialectorSqlite(ctx, cfg)

		require.NoError(t, err)
		assert.NotNil(t, dialector)
	})

	t.Run("success with file path", func(t *testing.T) {
		ctx := context.TestContext(nil)
		cfg := config.DbConfig{
			Type: DbTypeSqlite,
			Config: map[string]interface{}{
				"dsn": "/tmp/test.db",
			},
		}

		dialector, err := CreateDialectorSqlite(ctx, cfg)

		require.NoError(t, err)
		assert.NotNil(t, dialector)
	})

	t.Run("validation error missing dsn", func(t *testing.T) {
		ctx := context.TestContext(nil)
		cfg := config.DbConfig{
			Type:   DbTypeSqlite,
			Config: map[string]interface{}{},
		}

		dialector, err := CreateDialectorSqlite(ctx, cfg)

		require.Error(t, err)
		assert.Nil(t, dialector)
	})

	t.Run("validation error empty dsn", func(t *testing.T) {
		ctx := context.TestContext(nil)
		cfg := config.DbConfig{
			Type: DbTypeSqlite,
			Config: map[string]interface{}{
				"dsn": "",
			},
		}

		dialector, err := CreateDialectorSqlite(ctx, cfg)

		require.Error(t, err)
		assert.Nil(t, dialector)
	})

	t.Run("decode error invalid config type", func(t *testing.T) {
		ctx := context.TestContext(nil)
		cfg := config.DbConfig{
			Type: DbTypeSqlite,
			Config: map[string]interface{}{
				"dsn": 12345, // int instead of string
			},
		}

		dialector, err := CreateDialectorSqlite(ctx, cfg)

		require.Error(t, err)
		assert.Nil(t, dialector)
	})
}
