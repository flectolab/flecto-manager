package db

import (
	"errors"
	"testing"

	"github.com/flectolab/flecto-manager/config"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDemoTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(database.Models...)
	require.NoError(t, err)

	return db
}

func TestGetDemoCmd(t *testing.T) {
	ctx := appContext.TestContext(nil)
	cmd := GetDemoCmd(ctx)

	assert.Equal(t, "demo", cmd.Use)
	assert.Equal(t, "add demo data", cmd.Short)
}

func TestGetDemoRunFn_Success(t *testing.T) {
	db := setupDemoTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}
	ctx.Config.Page = config.PageConfig{
		SizeLimit:      1048576,
		TotalSizeLimit: 104857600,
	}

	oldNewDemoDB := NewDemoDB
	NewDemoDB = func(c *appContext.Context) (*gorm.DB, error) {
		return db, nil
	}
	defer func() { NewDemoDB = oldNewDemoDB }()

	cmd := GetDemoCmd(ctx)
	err := cmd.Execute()

	assert.NoError(t, err)

	// verify namespaces were created
	var namespaces []model.Namespace
	err = db.Find(&namespaces).Error
	assert.NoError(t, err)
	assert.Len(t, namespaces, 2)

	// verify projects were created
	var projects []model.Project
	err = db.Find(&projects).Error
	assert.NoError(t, err)
	assert.Len(t, projects, 6)
}

func TestGetDemoRunFn_DBError(t *testing.T) {
	ctx := appContext.TestContext(nil)

	oldNewDemoDB := NewDemoDB
	NewDemoDB = func(c *appContext.Context) (*gorm.DB, error) {
		return nil, errors.New("connection failed")
	}
	defer func() { NewDemoDB = oldNewDemoDB }()

	cmd := GetDemoCmd(ctx)
	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestDemoData_CreatesNamespaces(t *testing.T) {
	db := setupDemoTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}
	ctx.Config.Page = config.PageConfig{
		SizeLimit:      1048576,
		TotalSizeLimit: 104857600,
	}

	err := demoData(ctx, db)
	assert.NoError(t, err)

	var ns1 model.Namespace
	err = db.Where("namespace_code = ?", "ns1").First(&ns1).Error
	assert.NoError(t, err)
	assert.Equal(t, "Namespace 1", ns1.Name)

	var ns2 model.Namespace
	err = db.Where("namespace_code = ?", "ns2").First(&ns2).Error
	assert.NoError(t, err)
	assert.Equal(t, "Namespace 2", ns2.Name)
}

func TestDemoData_CreatesProjects(t *testing.T) {
	db := setupDemoTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}
	ctx.Config.Page = config.PageConfig{
		SizeLimit:      1048576,
		TotalSizeLimit: 104857600,
	}

	err := demoData(ctx, db)
	assert.NoError(t, err)

	// verify projects in namespace 1
	var projectsNs1 []model.Project
	err = db.Where("namespace_code = ?", "ns1").Find(&projectsNs1).Error
	assert.NoError(t, err)
	assert.Len(t, projectsNs1, 3)

	// verify projects in namespace 2
	var projectsNs2 []model.Project
	err = db.Where("namespace_code = ?", "ns2").Find(&projectsNs2).Error
	assert.NoError(t, err)
	assert.Len(t, projectsNs2, 3)
}

func TestDemoData_CreatesRedirects(t *testing.T) {
	db := setupDemoTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}
	ctx.Config.Page = config.PageConfig{
		SizeLimit:      1048576,
		TotalSizeLimit: 104857600,
	}

	err := demoData(ctx, db)
	assert.NoError(t, err)

	// verify redirects were created and published (39 redirects for /project/1 to /project/39)
	// drafts are deleted after publish, so we check the published redirects
	var redirects []model.Redirect
	err = db.Where("namespace_code = ? AND project_code = ?", "ns1", "prj1").Find(&redirects).Error
	assert.NoError(t, err)
	assert.Len(t, redirects, 39)
}

func TestDemoData_CreatesPage(t *testing.T) {
	db := setupDemoTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}
	ctx.Config.Page = config.PageConfig{
		SizeLimit:      1048576,
		TotalSizeLimit: 104857600,
	}

	err := demoData(ctx, db)
	assert.NoError(t, err)

	// verify page was created and published
	// drafts are deleted after publish, so we check the published pages
	var pages []model.Page
	err = db.Where("namespace_code = ? AND project_code = ?", "ns1", "prj1").Find(&pages).Error
	assert.NoError(t, err)
	assert.Len(t, pages, 1)
	assert.Equal(t, "/robots.txt", pages[0].Path)
}

func TestDemoData_DuplicateNamespaceError(t *testing.T) {
	db := setupDemoTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}
	ctx.Config.Page = config.PageConfig{
		SizeLimit:      1048576,
		TotalSizeLimit: 104857600,
	}

	// first call should succeed
	err := demoData(ctx, db)
	assert.NoError(t, err)

	// second call should fail due to duplicate namespace
	err = demoData(ctx, db)
	assert.Error(t, err)
}
