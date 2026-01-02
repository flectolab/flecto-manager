package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/config"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/jwt"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthServiceTest(t *testing.T) (*gomock.Controller, *mockFlectoRepository.MockUserRepository, *jwt.ServiceJWT, AuthService) {
	ctrl := gomock.NewController(t)
	mockUserRepo := mockFlectoRepository.NewMockUserRepository(ctrl)
	jwtService := jwt.NewServiceJWT(&config.JWTConfig{
		Secret:          "test-secret-key-32-bytes-long!!!",
		Issuer:          "test-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})
	ctx := appContext.TestContext(nil)
	svc := NewAuthService(ctx, mockUserRepo, jwtService)
	return ctrl, mockUserRepo, jwtService, svc
}

func TestNewAuthService(t *testing.T) {
	ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockUserRepo)
}

func TestAuthService_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		password := "testPassword123"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		user := &model.User{
			ID:       1,
			Username: "testuser",
			Password: string(hashedPassword),
			Active:   boolPtr(true),
		}

		req := &types.LoginRequest{
			Username: "testuser",
			Password: password,
		}

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(user, nil)

		mockUserRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, u *model.User) error {
				assert.NotEmpty(t, u.RefreshTokenHash)
				return nil
			})

		resultUser, tokens, err := svc.Login(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resultUser)
		assert.NotNil(t, tokens)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := &types.LoginRequest{
			Username: "unknownuser",
			Password: "password",
		}

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "unknownuser").
			Return(nil, gorm.ErrRecordNotFound)

		resultUser, tokens, err := svc.Login(ctx, req)

		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("database error on find", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := &types.LoginRequest{
			Username: "testuser",
			Password: "password",
		}
		dbErr := errors.New("database error")

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(nil, dbErr)

		resultUser, tokens, err := svc.Login(ctx, req)

		assert.Equal(t, dbErr, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("user inactive", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		user := &model.User{
			ID:       1,
			Username: "testuser",
			Password: "hashedpassword",
			Active:   boolPtr(false),
		}

		req := &types.LoginRequest{
			Username: "testuser",
			Password: "password",
		}

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(user, nil)

		resultUser, tokens, err := svc.Login(ctx, req)

		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("user has no password", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		user := &model.User{
			ID:       1,
			Username: "testuser",
			Password: "",
			Active:   boolPtr(true),
		}

		req := &types.LoginRequest{
			Username: "testuser",
			Password: "password",
		}

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(user, nil)

		resultUser, tokens, err := svc.Login(ctx, req)

		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("wrong password", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctPassword"), bcrypt.DefaultCost)

		user := &model.User{
			ID:       1,
			Username: "testuser",
			Password: string(hashedPassword),
			Active:   boolPtr(true),
		}

		req := &types.LoginRequest{
			Username: "testuser",
			Password: "wrongPassword",
		}

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(user, nil)

		resultUser, tokens, err := svc.Login(ctx, req)

		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("update error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		password := "testPassword123"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		user := &model.User{
			ID:       1,
			Username: "testuser",
			Password: string(hashedPassword),
			Active:   boolPtr(true),
		}

		req := &types.LoginRequest{
			Username: "testuser",
			Password: password,
		}
		updateErr := errors.New("update error")

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(user, nil)

		mockUserRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(updateErr)

		resultUser, tokens, err := svc.Login(ctx, req)

		assert.Equal(t, updateErr, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})
}

func TestAuthService_RefreshTokens(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, jwtService, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		user := &model.User{
			ID:       1,
			Username: "testuser",
			Active:   boolPtr(true),
		}

		// Generate initial tokens to get a valid refresh token
		tokenPair, _ := jwtService.GenerateTokenPair(user, types.AuthTypeBasic, nil, nil)
		user.RefreshTokenHash = jwt.HashToken(tokenPair.RefreshToken)

		claims := &jwt.Claims{
			UserID:    1,
			TokenType: types.TokenTypeRefresh,
		}

		// Setup mock for GetQuery to handle the update
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		db.AutoMigrate(&model.User{})
		db.Create(user)

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(user, nil)

		mockUserRepo.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.User{}))

		resultUser, tokens, err := svc.RefreshTokens(ctx, tokenPair.RefreshToken, claims)

		assert.NoError(t, err)
		assert.NotNil(t, resultUser)
		assert.NotNil(t, tokens)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("wrong token type", func(t *testing.T) {
		ctrl, _, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		claims := &jwt.Claims{
			UserID:    1,
			TokenType: types.TokenTypeAccess, // Wrong type
		}

		resultUser, tokens, err := svc.RefreshTokens(ctx, "some-token", claims)

		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		claims := &jwt.Claims{
			UserID:    999,
			TokenType: types.TokenTypeRefresh,
		}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		resultUser, tokens, err := svc.RefreshTokens(ctx, "some-token", claims)

		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("database error on find", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		claims := &jwt.Claims{
			UserID:    1,
			TokenType: types.TokenTypeRefresh,
		}
		dbErr := errors.New("database error")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, dbErr)

		resultUser, tokens, err := svc.RefreshTokens(ctx, "some-token", claims)

		assert.Equal(t, dbErr, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("user inactive", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		user := &model.User{
			ID:       1,
			Username: "testuser",
			Active:   boolPtr(false),
		}

		claims := &jwt.Claims{
			UserID:    1,
			TokenType: types.TokenTypeRefresh,
		}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(user, nil)

		resultUser, tokens, err := svc.RefreshTokens(ctx, "some-token", claims)

		assert.Equal(t, ErrUserInactive, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("invalid refresh token hash", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		user := &model.User{
			ID:               1,
			Username:         "testuser",
			Active:           boolPtr(true),
			RefreshTokenHash: "different-hash",
		}

		claims := &jwt.Claims{
			UserID:    1,
			TokenType: types.TokenTypeRefresh,
		}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(user, nil)

		resultUser, tokens, err := svc.RefreshTokens(ctx, "some-token", claims)

		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})

	t.Run("update error", func(t *testing.T) {
		ctrl, mockUserRepo, jwtService, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		user := &model.User{
			ID:       1,
			Username: "testuser",
			Active:   boolPtr(true),
		}

		// Generate initial tokens to get a valid refresh token
		tokenPair, _ := jwtService.GenerateTokenPair(user, types.AuthTypeBasic, nil, nil)
		user.RefreshTokenHash = jwt.HashToken(tokenPair.RefreshToken)

		claims := &jwt.Claims{
			UserID:    1,
			TokenType: types.TokenTypeRefresh,
		}

		// Setup mock for GetQuery to return a db that will fail (no table)
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(user, nil)

		mockUserRepo.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.User{}))

		resultUser, tokens, err := svc.RefreshTokens(ctx, tokenPair.RefreshToken, claims)

		assert.Error(t, err)
		assert.Nil(t, resultUser)
		assert.Nil(t, tokens)
	})
}

func TestAuthService_Logout(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Setup mock for GetQuery
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		db.AutoMigrate(&model.User{})
		db.Create(&model.User{
			ID:               1,
			Username:         "testuser",
			RefreshTokenHash: "some-hash",
		})

		mockUserRepo.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.User{}))

		err := svc.Logout(ctx, 1)

		assert.NoError(t, err)

		// Verify token hash was cleared
		var user model.User
		db.First(&user, 1)
		assert.Empty(t, user.RefreshTokenHash)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Setup mock for GetQuery with a db that will fail (no table)
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

		mockUserRepo.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.User{}))

		err := svc.Logout(ctx, 1)

		assert.Error(t, err)
	})
}

func TestAuthService_ToUserResponse(t *testing.T) {
	t.Run("converts user to response", func(t *testing.T) {
		ctrl, _, _, svc := setupAuthServiceTest(t)
		defer ctrl.Finish()

		user := &model.User{
			ID:        123,
			Username:  "testuser",
			Firstname: "Test",
			Lastname:  "User",
		}

		response := svc.ToUserResponse(user)

		assert.NotNil(t, response)
		assert.Equal(t, int64(123), response.ID)
		assert.Equal(t, "testuser", response.Username)
		assert.Equal(t, "Test", response.Firstname)
		assert.Equal(t, "User", response.Lastname)
	})
}