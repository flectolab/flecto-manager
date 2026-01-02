package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRepositoriesTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	return db
}

func TestNewRepositories(t *testing.T) {
	db := setupRepositoriesTestDB(t)

	repos := NewRepositories(db)

	assert.NotNil(t, repos)
	assert.NotNil(t, repos.Namespace)
	assert.NotNil(t, repos.Project)
	assert.NotNil(t, repos.User)
	assert.NotNil(t, repos.Role)
	assert.NotNil(t, repos.Redirect)
	assert.NotNil(t, repos.RedirectDraft)
	assert.NotNil(t, repos.Page)
	assert.NotNil(t, repos.PageDraft)
	assert.NotNil(t, repos.Agent)
	assert.NotNil(t, repos.Token)
}
