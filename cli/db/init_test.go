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

func setupInitTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(database.Models...)
	require.NoError(t, err)

	return db
}

func TestGetInitCmd(t *testing.T) {
	ctx := appContext.TestContext(nil)
	cmd := GetInitCmd(ctx)

	assert.Equal(t, "init", cmd.Use)
	assert.Equal(t, "init database", cmd.Short)
}

func TestGetInitRunFn_Success(t *testing.T) {
	db := setupInitTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	oldNewInitDB := NewInitDB
	NewInitDB = func(c *appContext.Context) (*gorm.DB, error) {
		return db, nil
	}
	defer func() { NewInitDB = oldNewInitDB }()

	cmd := GetInitCmd(ctx)
	err := cmd.Execute()

	assert.NoError(t, err)

	// verify admin user was created
	var user model.User
	err = db.Where("username = ?", "admin").First(&user).Error
	assert.NoError(t, err)
	assert.Equal(t, "admin", user.Username)
	assert.Equal(t, "Admin", user.Firstname)
	assert.Equal(t, "Admin", user.Lastname)
	assert.True(t, *user.Active)

	// verify admin role was created (filter by type to distinguish from user's personal role)
	var role model.Role
	err = db.Where("code = ? AND type = ?", "admin", model.RoleTypeRole).First(&role).Error
	assert.NoError(t, err)
	assert.Equal(t, "admin", role.Code)
	assert.Equal(t, model.RoleTypeRole, role.Type)

	// verify user has admin role
	var userRole model.UserRole
	err = db.Where("user_id = ? AND role_id = ?", user.ID, role.ID).First(&userRole).Error
	assert.NoError(t, err)
}

func TestGetInitRunFn_DBError(t *testing.T) {
	ctx := appContext.TestContext(nil)

	oldNewInitDB := NewInitDB
	NewInitDB = func(c *appContext.Context) (*gorm.DB, error) {
		return nil, errors.New("connection failed")
	}
	defer func() { NewInitDB = oldNewInitDB }()

	cmd := GetInitCmd(ctx)
	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestInitData_CreatesAdminUserWithCorrectPassword(t *testing.T) {
	db := setupInitTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	err := initData(ctx, db)
	assert.NoError(t, err)

	var user model.User
	err = db.Where("username = ?", "admin").First(&user).Error
	assert.NoError(t, err)

	// password should be bcrypt hashed
	assert.NotEmpty(t, user.Password)
	assert.NotEqual(t, "admin", user.Password)
}

func TestInitData_CreatesAdminRoleWithCorrectPermissions(t *testing.T) {
	db := setupInitTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	err := initData(ctx, db)
	assert.NoError(t, err)

	// filter by type to distinguish from user's personal role
	var role model.Role
	err = db.Preload("Resources").Preload("Admin").Where("code = ? AND type = ?", "admin", model.RoleTypeRole).First(&role).Error
	assert.NoError(t, err)

	// verify resource permissions
	require.Len(t, role.Resources, 1)
	assert.Equal(t, "*", role.Resources[0].Namespace)
	assert.Equal(t, "*", role.Resources[0].Project)
	assert.Equal(t, model.ActionAll, role.Resources[0].Action)
	assert.Equal(t, model.ResourceTypeAll, role.Resources[0].Resource)

	// verify admin permissions
	require.Len(t, role.Admin, 1)
	assert.Equal(t, model.AdminSectionAll, role.Admin[0].Section)
	assert.Equal(t, model.ActionAll, role.Admin[0].Action)
}

func TestInitData_DuplicateUserError(t *testing.T) {
	db := setupInitTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	// first call should succeed
	err := initData(ctx, db)
	assert.NoError(t, err)

	// second call should fail due to duplicate user
	err = initData(ctx, db)
	assert.Error(t, err)
}
