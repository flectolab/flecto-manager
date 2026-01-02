package repository

import (
	"context"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTokenRepositoryTest(t *testing.T) (*gorm.DB, TokenRepository) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Token{})
	assert.NoError(t, err)

	repo := NewTokenRepository(db)
	return db, repo
}

func TestNewTokenRepository(t *testing.T) {
	_, repo := setupTokenRepositoryTest(t)
	assert.NotNil(t, repo)
}

func TestTokenRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		token := &model.Token{
			Name:      "test-token",
			TokenHash: "hash123",
		}

		err := repo.Create(ctx, token)
		assert.NoError(t, err)
		assert.NotZero(t, token.ID)

		var saved model.Token
		db.First(&saved, token.ID)
		assert.Equal(t, "test-token", saved.Name)
	})

	t.Run("duplicate name", func(t *testing.T) {
		_, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		token1 := &model.Token{Name: "duplicate", TokenHash: "hash1"}
		err := repo.Create(ctx, token1)
		assert.NoError(t, err)

		token2 := &model.Token{Name: "duplicate", TokenHash: "hash2"}
		err = repo.Create(ctx, token2)
		assert.Error(t, err)
	})

	t.Run("duplicate hash", func(t *testing.T) {
		_, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		token1 := &model.Token{Name: "token1", TokenHash: "samehash"}
		err := repo.Create(ctx, token1)
		assert.NoError(t, err)

		token2 := &model.Token{Name: "token2", TokenHash: "samehash"}
		err = repo.Create(ctx, token2)
		assert.Error(t, err)
	})
}

func TestTokenRepository_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		token := &model.Token{Name: "to-delete", TokenHash: "hash"}
		db.Create(token)

		err := repo.Delete(ctx, token.ID)
		assert.NoError(t, err)

		var count int64
		db.Model(&model.Token{}).Where("id = ?", token.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("non-existent", func(t *testing.T) {
		_, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		err := repo.Delete(ctx, 999)
		assert.NoError(t, err) // GORM doesn't error on non-existent delete
	})
}

func TestTokenRepository_FindByID(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		token := &model.Token{Name: "test", TokenHash: "hash"}
		db.Create(token)

		result, err := repo.FindByID(ctx, token.ID)
		assert.NoError(t, err)
		assert.Equal(t, token.ID, result.ID)
		assert.Equal(t, "test", result.Name)
	})

	t.Run("not found", func(t *testing.T) {
		_, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		result, err := repo.FindByID(ctx, 999)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestTokenRepository_FindByName(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		token := &model.Token{Name: "my-token", TokenHash: "hash"}
		db.Create(token)

		result, err := repo.FindByName(ctx, "my-token")
		assert.NoError(t, err)
		assert.Equal(t, token.ID, result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		result, err := repo.FindByName(ctx, "unknown")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestTokenRepository_FindByHash(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		token := &model.Token{Name: "test", TokenHash: "uniquehash123"}
		db.Create(token)

		result, err := repo.FindByHash(ctx, "uniquehash123")
		assert.NoError(t, err)
		assert.Equal(t, token.ID, result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		result, err := repo.FindByHash(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestTokenRepository_FindAll(t *testing.T) {
	t.Run("returns all tokens", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		db.Create(&model.Token{Name: "token1", TokenHash: "hash1"})
		db.Create(&model.Token{Name: "token2", TokenHash: "hash2"})

		result, err := repo.FindAll(ctx)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("empty result", func(t *testing.T) {
		_, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		result, err := repo.FindAll(ctx)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestTokenRepository_SearchPaginate(t *testing.T) {
	t.Run("with pagination", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		for i := 0; i < 15; i++ {
			db.Create(&model.Token{Name: "token" + string(rune('A'+i)), TokenHash: "hash" + string(rune('A'+i))})
		}

		result, total, err := repo.SearchPaginate(ctx, nil, 5, 0)
		assert.NoError(t, err)
		assert.Len(t, result, 5)
		assert.Equal(t, int64(15), total)
	})

	t.Run("with offset", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		for i := 0; i < 10; i++ {
			db.Create(&model.Token{Name: "token" + string(rune('A'+i)), TokenHash: "hash" + string(rune('A'+i))})
		}

		result, total, err := repo.SearchPaginate(ctx, nil, 5, 5)
		assert.NoError(t, err)
		assert.Len(t, result, 5)
		assert.Equal(t, int64(10), total)
	})

	t.Run("with custom query", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		db.Create(&model.Token{Name: "api-token", TokenHash: "hash1"})
		db.Create(&model.Token{Name: "web-token", TokenHash: "hash2"})

		query := db.Model(&model.Token{}).Where("name LIKE ?", "api%")
		result, total, err := repo.SearchPaginate(ctx, query, 10, 0)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "api-token", result[0].Name)
	})

	t.Run("zero limit returns all", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			db.Create(&model.Token{Name: "token" + string(rune('A'+i)), TokenHash: "hash" + string(rune('A'+i))})
		}

		result, total, err := repo.SearchPaginate(ctx, nil, 0, 0)
		assert.NoError(t, err)
		assert.Len(t, result, 5)
		assert.Equal(t, int64(5), total)
	})
}

func TestTokenRepository_GetTx(t *testing.T) {
	_, repo := setupTokenRepositoryTest(t)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)
}

func TestTokenRepository_GetQuery(t *testing.T) {
	_, repo := setupTokenRepositoryTest(t)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)
}

func TestTokenRepository_ExpiresAt(t *testing.T) {
	t.Run("with expiration", func(t *testing.T) {
		_, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		expiresAt := time.Now().Add(time.Hour)
		token := &model.Token{
			Name:      "expiring-token",
			TokenHash: "hash",
			ExpiresAt: &expiresAt,
		}
		err := repo.Create(ctx, token)
		assert.NoError(t, err)

		result, err := repo.FindByID(ctx, token.ID)
		assert.NoError(t, err)
		assert.NotNil(t, result.ExpiresAt)
	})

	t.Run("without expiration", func(t *testing.T) {
		db, repo := setupTokenRepositoryTest(t)
		ctx := context.Background()

		token := &model.Token{
			Name:      "permanent-token",
			TokenHash: "hash",
		}
		err := repo.Create(ctx, token)
		assert.NoError(t, err)

		result, err := repo.FindByID(ctx, token.ID)
		assert.NoError(t, err)
		assert.Nil(t, result.ExpiresAt)

		// Also verify directly from DB
		var dbToken model.Token
		db.First(&dbToken, token.ID)
		assert.Nil(t, dbToken.ExpiresAt)
	})
}
