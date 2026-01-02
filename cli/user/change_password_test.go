package user

import (
	"errors"
	"testing"

	"github.com/flectolab/flecto-manager/config"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupChangePasswordTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(database.Models...)
	require.NoError(t, err)

	return db
}

func createTestUser(t *testing.T, db *gorm.DB, username, password string) *model.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	require.NoError(t, err)

	user := &model.User{
		Username:  username,
		Password:  string(hashedPassword),
		Firstname: "Test",
		Lastname:  "User",
		Active:    types.Ptr(true),
	}
	err = db.Create(user).Error
	require.NoError(t, err)

	return user
}

func TestGetChangePasswordCmd(t *testing.T) {
	ctx := appContext.TestContext(nil)
	cmd := GetChangePasswordCmd(ctx)

	assert.Equal(t, "change-password", cmd.Use)
	assert.Equal(t, "change password for user", cmd.Short)
}

func TestGetChangePasswordCmd_HasFlags(t *testing.T) {
	ctx := appContext.TestContext(nil)
	cmd := GetChangePasswordCmd(ctx)

	// verify username flag
	usernameFlag := cmd.Flags().Lookup("username")
	assert.NotNil(t, usernameFlag)
	assert.Equal(t, "u", usernameFlag.Shorthand)

	// verify password flag
	passwordFlag := cmd.Flags().Lookup("password")
	assert.NotNil(t, passwordFlag)
	assert.Equal(t, "p", passwordFlag.Shorthand)
}

func TestGetChangePasswordRunFn_Success(t *testing.T) {
	db := setupChangePasswordTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	// create test user
	testUser := createTestUser(t, db, "testuser", "oldpassword")

	oldNewChangePasswordDB := NewChangePasswordDB
	NewChangePasswordDB = func(c *appContext.Context) (*gorm.DB, error) {
		return db, nil
	}
	defer func() { NewChangePasswordDB = oldNewChangePasswordDB }()

	cmd := GetChangePasswordCmd(ctx)
	cmd.SetArgs([]string{"-u", "testuser", "-p", "newpassword"})

	err := cmd.Execute()
	assert.NoError(t, err)

	// verify password was changed
	var updatedUser model.User
	err = db.First(&updatedUser, testUser.ID).Error
	assert.NoError(t, err)
	assert.NotEqual(t, testUser.Password, updatedUser.Password)

	// verify new password works
	err = bcrypt.CompareHashAndPassword([]byte(updatedUser.Password), []byte("newpassword"))
	assert.NoError(t, err)
}

func TestGetChangePasswordRunFn_DBError(t *testing.T) {
	ctx := appContext.TestContext(nil)

	oldNewChangePasswordDB := NewChangePasswordDB
	NewChangePasswordDB = func(c *appContext.Context) (*gorm.DB, error) {
		return nil, errors.New("connection failed")
	}
	defer func() { NewChangePasswordDB = oldNewChangePasswordDB }()

	cmd := GetChangePasswordCmd(ctx)
	cmd.SetArgs([]string{"-u", "testuser", "-p", "newpassword"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestGetChangePasswordRunFn_EmptyUsername(t *testing.T) {
	db := setupChangePasswordTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	oldNewChangePasswordDB := NewChangePasswordDB
	NewChangePasswordDB = func(c *appContext.Context) (*gorm.DB, error) {
		return db, nil
	}
	defer func() { NewChangePasswordDB = oldNewChangePasswordDB }()

	cmd := GetChangePasswordCmd(ctx)
	cmd.SetArgs([]string{"-u", "", "-p", "newpassword"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username and password cannot be empty")
}

func TestGetChangePasswordRunFn_EmptyPassword(t *testing.T) {
	db := setupChangePasswordTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	oldNewChangePasswordDB := NewChangePasswordDB
	NewChangePasswordDB = func(c *appContext.Context) (*gorm.DB, error) {
		return db, nil
	}
	defer func() { NewChangePasswordDB = oldNewChangePasswordDB }()

	cmd := GetChangePasswordCmd(ctx)
	cmd.SetArgs([]string{"-u", "testuser", "-p", ""})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username and password cannot be empty")
}

func TestGetChangePasswordRunFn_BothEmpty(t *testing.T) {
	db := setupChangePasswordTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	oldNewChangePasswordDB := NewChangePasswordDB
	NewChangePasswordDB = func(c *appContext.Context) (*gorm.DB, error) {
		return db, nil
	}
	defer func() { NewChangePasswordDB = oldNewChangePasswordDB }()

	cmd := GetChangePasswordCmd(ctx)
	cmd.SetArgs([]string{"-u", "", "-p", ""})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username and password cannot be empty")
}

func TestGetChangePasswordRunFn_UserNotFound(t *testing.T) {
	db := setupChangePasswordTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	oldNewChangePasswordDB := NewChangePasswordDB
	NewChangePasswordDB = func(c *appContext.Context) (*gorm.DB, error) {
		return db, nil
	}
	defer func() { NewChangePasswordDB = oldNewChangePasswordDB }()

	cmd := GetChangePasswordCmd(ctx)
	cmd.SetArgs([]string{"-u", "nonexistent", "-p", "newpassword"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestGetChangePasswordRunFn_PasswordIsHashed(t *testing.T) {
	db := setupChangePasswordTestDB(t)
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT = config.JWTConfig{
		Secret:          "test-secret-key-for-jwt-minimum-32-chars",
		Issuer:          "test-issuer",
		AccessTokenTTL:  900,
		RefreshTokenTTL: 86400,
		HeaderName:      "Authorization",
	}

	testUser := createTestUser(t, db, "hashuser", "oldpassword")

	oldNewChangePasswordDB := NewChangePasswordDB
	NewChangePasswordDB = func(c *appContext.Context) (*gorm.DB, error) {
		return db, nil
	}
	defer func() { NewChangePasswordDB = oldNewChangePasswordDB }()

	newPassword := "mynewsecurepassword"
	cmd := GetChangePasswordCmd(ctx)
	cmd.SetArgs([]string{"-u", "hashuser", "-p", newPassword})

	err := cmd.Execute()
	assert.NoError(t, err)

	// verify password is hashed, not stored in plain text
	var updatedUser model.User
	err = db.First(&updatedUser, testUser.ID).Error
	assert.NoError(t, err)
	assert.NotEqual(t, newPassword, updatedUser.Password)

	// verify it's a valid bcrypt hash
	err = bcrypt.CompareHashAndPassword([]byte(updatedUser.Password), []byte(newPassword))
	assert.NoError(t, err)
}
