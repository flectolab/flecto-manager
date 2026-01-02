package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/jwt"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type tokenServiceMocks struct {
	ctrl      *gomock.Controller
	tokenRepo *mockFlectoRepository.MockTokenRepository
	roleRepo  *mockFlectoRepository.MockRoleRepository
}

func setupTokenServiceTest(t *testing.T) (*tokenServiceMocks, TokenService) {
	ctrl := gomock.NewController(t)
	mocks := &tokenServiceMocks{
		ctrl:      ctrl,
		tokenRepo: mockFlectoRepository.NewMockTokenRepository(ctrl),
		roleRepo:  mockFlectoRepository.NewMockRoleRepository(ctrl),
	}
	svc := NewTokenService(appContext.TestContext(nil), mocks.tokenRepo, mocks.roleRepo)
	return mocks, svc
}

func TestNewTokenService(t *testing.T) {
	mocks, svc := setupTokenServiceTest(t)
	defer mocks.ctrl.Finish()

	assert.NotNil(t, svc)
}

func TestTokenService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedToken := &model.Token{
			ID:   1,
			Name: "test-token",
		}

		mocks.tokenRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(expectedToken, nil)

		result, err := svc.GetByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedToken, result)
	})

	t.Run("not found", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.tokenRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetByID(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("generic error", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.tokenRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.GetByID(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestTokenService_GetByName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedToken := &model.Token{
			ID:   1,
			Name: "test-token",
		}

		mocks.tokenRepo.EXPECT().
			FindByName(ctx, "test-token").
			Return(expectedToken, nil)

		result, err := svc.GetByName(ctx, "test-token")

		assert.NoError(t, err)
		assert.Equal(t, expectedToken, result)
	})

	t.Run("not found", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.tokenRepo.EXPECT().
			FindByName(ctx, "unknown").
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetByName(ctx, "unknown")

		assert.Error(t, err)
		assert.Equal(t, ErrTokenNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("generic error", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.tokenRepo.EXPECT().
			FindByName(ctx, "test").
			Return(nil, expectedErr)

		result, err := svc.GetByName(ctx, "test")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestTokenService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedTokens := []model.Token{
			{ID: 1, Name: "token1"},
			{ID: 2, Name: "token2"},
		}

		mocks.tokenRepo.EXPECT().
			FindAll(ctx).
			Return(expectedTokens, nil)

		result, err := svc.GetAll(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expectedTokens, result)
	})
}

func TestTokenService_SearchPaginate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &types.PaginationInput{
			Limit:  &limit,
			Offset: &offset,
		}
		expectedTokens := []model.Token{
			{ID: 1, Name: "token1"},
			{ID: 2, Name: "token2"},
		}

		mocks.tokenRepo.EXPECT().
			SearchPaginate(ctx, nil, 10, 5).
			Return(expectedTokens, int64(20), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 20, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 5, result.Offset)
		assert.Len(t, result.Items, 2)
	})

	t.Run("error", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedErr := errors.New("search error")

		mocks.tokenRepo.EXPECT().
			SearchPaginate(ctx, nil, types.DefaultLimit, types.DefaultOffset).
			Return(nil, int64(0), expectedErr)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestTokenService_ValidateToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		plainToken := "flecto_testtoken123456789012345678901234"
		tokenHash := jwt.HashToken(plainToken)
		token := &model.Token{
			ID:        1,
			Name:      "testtoken",
			TokenHash: tokenHash,
		}
		role := &model.Role{
			ID:   1,
			Code: "token_testtoken",
			Type: model.RoleTypeToken,
			Resources: []model.ResourcePermission{
				{ID: 1, Namespace: "ns1", Action: model.ActionRead},
			},
		}

		mocks.tokenRepo.EXPECT().
			FindByHash(ctx, tokenHash).
			Return(token, nil)

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "token_testtoken", model.RoleTypeToken).
			Return(role, nil)

		resultToken, permissions, err := svc.ValidateToken(ctx, plainToken)

		assert.NoError(t, err)
		assert.Equal(t, token, resultToken)
		assert.Len(t, permissions.Resources, 1)
	})

	t.Run("invalid prefix", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		plainToken := "invalid_token"

		resultToken, permissions, err := svc.ValidateToken(ctx, plainToken)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidToken, err)
		assert.Nil(t, resultToken)
		assert.Nil(t, permissions)
	})

	t.Run("token too short", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		plainToken := "flect"

		resultToken, permissions, err := svc.ValidateToken(ctx, plainToken)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidToken, err)
		assert.Nil(t, resultToken)
		assert.Nil(t, permissions)
	})

	t.Run("token not found", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		plainToken := "flecto_unknowntoken12345678901234567890"
		tokenHash := jwt.HashToken(plainToken)

		mocks.tokenRepo.EXPECT().
			FindByHash(ctx, tokenHash).
			Return(nil, gorm.ErrRecordNotFound)

		resultToken, permissions, err := svc.ValidateToken(ctx, plainToken)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidToken, err)
		assert.Nil(t, resultToken)
		assert.Nil(t, permissions)
	})

	t.Run("token expired", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		plainToken := "flecto_expiredtoken1234567890123456789"
		tokenHash := jwt.HashToken(plainToken)
		expiredTime := time.Now().Add(-time.Hour)
		token := &model.Token{
			ID:        1,
			Name:      "expiredtoken",
			TokenHash: tokenHash,
			ExpiresAt: &expiredTime,
		}

		mocks.tokenRepo.EXPECT().
			FindByHash(ctx, tokenHash).
			Return(token, nil)

		resultToken, permissions, err := svc.ValidateToken(ctx, plainToken)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenExpired, err)
		assert.Nil(t, resultToken)
		assert.Nil(t, permissions)
	})

	t.Run("role not found returns empty permissions", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		plainToken := "flecto_testtoken123456789012345678901234"
		tokenHash := jwt.HashToken(plainToken)
		token := &model.Token{
			ID:        1,
			Name:      "testtoken",
			TokenHash: tokenHash,
		}

		mocks.tokenRepo.EXPECT().
			FindByHash(ctx, tokenHash).
			Return(token, nil)

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "token_testtoken", model.RoleTypeToken).
			Return(nil, gorm.ErrRecordNotFound)

		resultToken, permissions, err := svc.ValidateToken(ctx, plainToken)

		assert.NoError(t, err)
		assert.Equal(t, token, resultToken)
		assert.Empty(t, permissions.Resources)
		assert.Empty(t, permissions.Admin)
	})

	t.Run("find by hash generic error", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		plainToken := "flecto_testtoken123456789012345678901234"
		tokenHash := jwt.HashToken(plainToken)
		expectedErr := errors.New("database error")

		mocks.tokenRepo.EXPECT().
			FindByHash(ctx, tokenHash).
			Return(nil, expectedErr)

		resultToken, permissions, err := svc.ValidateToken(ctx, plainToken)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resultToken)
		assert.Nil(t, permissions)
	})

	t.Run("find role generic error", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		plainToken := "flecto_testtoken123456789012345678901234"
		tokenHash := jwt.HashToken(plainToken)
		token := &model.Token{
			ID:        1,
			Name:      "testtoken",
			TokenHash: tokenHash,
		}
		expectedErr := errors.New("database error")

		mocks.tokenRepo.EXPECT().
			FindByHash(ctx, tokenHash).
			Return(token, nil)

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "token_testtoken", model.RoleTypeToken).
			Return(nil, expectedErr)

		resultToken, permissions, err := svc.ValidateToken(ctx, plainToken)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, resultToken)
		assert.Nil(t, permissions)
	})
}

func TestTokenService_GetRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		token := &model.Token{
			ID:   1,
			Name: "testtoken",
		}
		expectedRole := &model.Role{
			ID:   1,
			Code: "token_testtoken",
			Type: model.RoleTypeToken,
		}

		mocks.tokenRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(token, nil)

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "token_testtoken", model.RoleTypeToken).
			Return(expectedRole, nil)

		result, err := svc.GetRole(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedRole, result)
	})

	t.Run("token not found", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.tokenRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetRole(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("role not found", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		token := &model.Token{
			ID:   1,
			Name: "testtoken",
		}

		mocks.tokenRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(token, nil)

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "token_testtoken", model.RoleTypeToken).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetRole(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("find token generic error", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.tokenRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.GetRole(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("find role generic error", func(t *testing.T) {
		mocks, svc := setupTokenServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		token := &model.Token{
			ID:   1,
			Name: "testtoken",
		}
		expectedErr := errors.New("database error")

		mocks.tokenRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(token, nil)

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "token_testtoken", model.RoleTypeToken).
			Return(nil, expectedErr)

		result, err := svc.GetRole(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestTokenService_GetTx(t *testing.T) {
	mocks, svc := setupTokenServiceTest(t)
	defer mocks.ctrl.Finish()

	ctx := context.Background()
	mocks.tokenRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestTokenService_GetQuery(t *testing.T) {
	mocks, svc := setupTokenServiceTest(t)
	defer mocks.ctrl.Finish()

	ctx := context.Background()
	mocks.tokenRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}

func TestParseDateTime(t *testing.T) {
	t.Run("valid RFC3339", func(t *testing.T) {
		result, err := parseDateTime("2024-01-15T10:30:00Z")
		assert.NoError(t, err)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.January, result.Month())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := parseDateTime("2024-01-15")
		assert.Error(t, err)
	})
}

// Integration tests

func setupTokenServiceIntegrationTest(t *testing.T) (*gorm.DB, TokenService) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Token{}, &model.Role{}, &model.ResourcePermission{}, &model.AdminPermission{})
	assert.NoError(t, err)

	tokenRepo := repository.NewTokenRepository(db)
	roleRepo := repository.NewRoleRepository(db)

	svc := NewTokenService(appContext.TestContext(nil), tokenRepo, roleRepo)
	return db, svc
}

func TestTokenService_Create_Integration(t *testing.T) {
	t.Run("success without expiration", func(t *testing.T) {
		_, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		token, plainToken, err := svc.Create(ctx, "test-token", nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, "test-token", token.Name)
		assert.True(t, strings.HasPrefix(plainToken, model.TokenPrefix))
		assert.Nil(t, token.ExpiresAt)
	})

	t.Run("success with expiration", func(t *testing.T) {
		_, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		expiresAt := "2025-12-31T23:59:59Z"
		token, plainToken, err := svc.Create(ctx, "test-token", &expiresAt, nil)

		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.True(t, strings.HasPrefix(plainToken, model.TokenPrefix))
		assert.NotNil(t, token.ExpiresAt)
	})

	t.Run("creates personal role", func(t *testing.T) {
		db, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		token, _, err := svc.Create(ctx, "test-token", nil, nil)
		assert.NoError(t, err)

		var role model.Role
		err = db.Where("code = ? AND type = ?", token.GetRoleCode(), model.RoleTypeToken).First(&role).Error
		assert.NoError(t, err)
		assert.Equal(t, "token_test-token", role.Code)
		assert.Equal(t, model.RoleTypeToken, role.Type)
	})

	t.Run("name too long", func(t *testing.T) {
		_, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		longName := strings.Repeat("a", model.TokenNameMaxLength+1)
		token, plainToken, err := svc.Create(ctx, longName, nil, nil)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenNameTooLong, err)
		assert.Nil(t, token)
		assert.Empty(t, plainToken)
	})

	t.Run("duplicate name", func(t *testing.T) {
		_, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		_, _, err := svc.Create(ctx, "duplicate", nil, nil)
		assert.NoError(t, err)

		_, _, err = svc.Create(ctx, "duplicate", nil, nil)
		assert.Error(t, err)
		assert.Equal(t, ErrTokenAlreadyExists, err)
	})

	t.Run("invalid expiration format", func(t *testing.T) {
		_, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		expiresAt := "invalid-date"
		token, plainToken, err := svc.Create(ctx, "test-token", &expiresAt, nil)

		assert.Error(t, err)
		assert.Nil(t, token)
		assert.Empty(t, plainToken)
	})

	t.Run("empty expiration string is treated as no expiration", func(t *testing.T) {
		_, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		expiresAt := ""
		token, _, err := svc.Create(ctx, "test-token", &expiresAt, nil)

		assert.NoError(t, err)
		assert.Nil(t, token.ExpiresAt)
	})

	t.Run("creates token with permissions", func(t *testing.T) {
		db, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Resource: model.ResourceTypeRedirect, Action: model.ActionRead},
				{Namespace: "*", Project: "*", Resource: model.ResourceTypePage, Action: model.ActionWrite},
			},
			Admin: []model.AdminPermission{
				{Section: model.AdminSectionUsers, Action: model.ActionRead},
			},
		}

		token, _, err := svc.Create(ctx, "test-token-with-perms", nil, permissions)
		assert.NoError(t, err)

		// Get the role
		var role model.Role
		err = db.Where("code = ? AND type = ?", token.GetRoleCode(), model.RoleTypeToken).First(&role).Error
		assert.NoError(t, err)

		// Check resource permissions
		var resourcePerms []model.ResourcePermission
		err = db.Where("role_id = ?", role.ID).Find(&resourcePerms).Error
		assert.NoError(t, err)
		assert.Len(t, resourcePerms, 2)

		// Check admin permissions
		var adminPerms []model.AdminPermission
		err = db.Where("role_id = ?", role.ID).Find(&adminPerms).Error
		assert.NoError(t, err)
		assert.Len(t, adminPerms, 1)
		assert.Equal(t, model.AdminSectionUsers, adminPerms[0].Section)
	})
}

func TestTokenService_Delete_Integration(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		token, _, err := svc.Create(ctx, "test-token", nil, nil)
		assert.NoError(t, err)

		result, err := svc.Delete(ctx, token.ID)
		assert.NoError(t, err)
		assert.True(t, result)

		// Verify token is deleted
		var count int64
		db.Model(&model.Token{}).Where("id = ?", token.ID).Count(&count)
		assert.Equal(t, int64(0), count)

		// Verify role is deleted
		db.Model(&model.Role{}).Where("code = ? AND type = ?", token.GetRoleCode(), model.RoleTypeToken).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("deletes associated permissions", func(t *testing.T) {
		db, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		token, _, err := svc.Create(ctx, "test-token", nil, nil)
		assert.NoError(t, err)

		// Get the role
		var role model.Role
		err = db.Where("code = ? AND type = ?", token.GetRoleCode(), model.RoleTypeToken).First(&role).Error
		assert.NoError(t, err)

		// Add permissions
		err = db.Create(&model.ResourcePermission{RoleID: role.ID, Namespace: "ns1", Action: model.ActionRead}).Error
		assert.NoError(t, err)
		err = db.Create(&model.AdminPermission{RoleID: role.ID, Section: model.AdminSectionUsers, Action: model.ActionRead}).Error
		assert.NoError(t, err)

		// Delete token
		_, err = svc.Delete(ctx, token.ID)
		assert.NoError(t, err)

		// Verify permissions are deleted
		var resourceCount, adminCount int64
		db.Model(&model.ResourcePermission{}).Where("role_id = ?", role.ID).Count(&resourceCount)
		db.Model(&model.AdminPermission{}).Where("role_id = ?", role.ID).Count(&adminCount)
		assert.Equal(t, int64(0), resourceCount)
		assert.Equal(t, int64(0), adminCount)
	})

	t.Run("token not found", func(t *testing.T) {
		_, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		result, err := svc.Delete(ctx, 999)
		assert.Error(t, err)
		assert.Equal(t, ErrTokenNotFound, err)
		assert.False(t, result)
	})
}

func TestTokenService_ValidateToken_Integration(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		token, plainToken, err := svc.Create(ctx, "test-token", nil, nil)
		assert.NoError(t, err)

		resultToken, permissions, err := svc.ValidateToken(ctx, plainToken)

		assert.NoError(t, err)
		assert.Equal(t, token.ID, resultToken.ID)
		assert.NotNil(t, permissions)
	})

	t.Run("expired token", func(t *testing.T) {
		db, svc := setupTokenServiceIntegrationTest(t)
		ctx := context.Background()

		// Create a token that has already expired
		expiresAt := "2020-01-01T00:00:00Z"
		token, plainToken, err := svc.Create(ctx, "expired-token", &expiresAt, nil)
		assert.NoError(t, err)

		// Manually update to past date since validation happens after creation
		pastTime := time.Now().Add(-time.Hour)
		db.Model(token).Update("expires_at", pastTime)

		_, _, err = svc.ValidateToken(ctx, plainToken)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenExpired, err)
	})
}
