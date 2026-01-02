package service

import (
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/config"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupServicesTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	return db
}

func setupServicesTest(t *testing.T) (*appContext.Context, *repository.Repositories, *jwt.ServiceJWT) {
	db := setupServicesTestDB(t)
	ctx := appContext.TestContext(nil)
	repos := repository.NewRepositories(db)
	jwtService := jwt.NewServiceJWT(&config.JWTConfig{
		Secret:          "test-secret-key-32-bytes-long!!!",
		Issuer:          "test-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})
	return ctx, repos, jwtService
}

func TestNewServices(t *testing.T) {
	ctx, repos, jwtService := setupServicesTest(t)

	services := NewServices(ctx, repos, jwtService)

	assert.NotNil(t, services)
	assert.NotNil(t, services.Namespace)
	assert.NotNil(t, services.Project)
	assert.NotNil(t, services.User)
	assert.NotNil(t, services.Auth)
	assert.NotNil(t, services.Role)
	assert.NotNil(t, services.Token)
	assert.NotNil(t, services.Redirect)
	assert.NotNil(t, services.RedirectDraft)
	assert.NotNil(t, services.RedirectImport)
	assert.NotNil(t, services.Page)
	assert.NotNil(t, services.PageDraft)
	assert.NotNil(t, services.Agent)
	assert.NotNil(t, services.ProjectDashboard)
}
