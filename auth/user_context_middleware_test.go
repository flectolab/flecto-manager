package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/jwt"
	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type middlewareMocks struct {
	ctrl         *gomock.Controller
	userService  *mockFlectoService.MockUserService
	roleService  *mockFlectoService.MockRoleService
	tokenService *mockFlectoService.MockTokenService
}

func setupMiddlewareMocks(t *testing.T) (*middlewareMocks, *config.JWTConfig) {
	ctrl := gomock.NewController(t)
	mocks := &middlewareMocks{
		ctrl:         ctrl,
		userService:  mockFlectoService.NewMockUserService(ctrl),
		roleService:  mockFlectoService.NewMockRoleService(ctrl),
		tokenService: mockFlectoService.NewMockTokenService(ctrl),
	}
	jwtConfig := &config.JWTConfig{
		Secret:          "test-secret-key",
		HeaderName:      "Authorization",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	}
	return mocks, jwtConfig
}

func TestUserCtxAuthMiddleware_MissingHeader(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid Authorization header")
}

func TestUserCtxAuthMiddleware_InvalidBearerFormat(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid Authorization header")
}

func TestUserCtxAuthMiddleware_ShortHeader(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bear")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid Authorization header")
}

func TestUserCtxAuthMiddleware_APIToken_Valid(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	plainToken := "flecto_testtoken123456789012345678901234"
	token := &model.Token{
		ID:   1,
		Name: "test-api-token",
	}
	permissions := &model.SubjectPermissions{
		Resources: []model.ResourcePermission{
			{Namespace: "ns1", Action: model.ActionRead},
		},
	}

	mocks.tokenService.EXPECT().
		ValidateToken(gomock.Any(), plainToken).
		Return(token, permissions, nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+plainToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)

	var userCtx *UserContext
	handler := middleware(func(c echo.Context) error {
		userCtx = GetUser(c.Request().Context())
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.NotNil(t, userCtx)
	assert.Equal(t, int64(0), userCtx.UserID)
	assert.Equal(t, "test-api-token", userCtx.Username)
	assert.Equal(t, types.AuthTypeToken, userCtx.AuthType)
	assert.Len(t, userCtx.SubjectPermissions.Resources, 1)
}

func TestUserCtxAuthMiddleware_APIToken_Invalid(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	plainToken := "flecto_invalidtoken"

	mocks.tokenService.EXPECT().
		ValidateToken(gomock.Any(), plainToken).
		Return(nil, nil, errors.New("invalid token"))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+plainToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API token")
}

func TestUserCtxAuthMiddleware_JWT_Valid(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	// Generate a valid JWT token
	jwtService := jwt.NewServiceJWT(jwtConfig)
	user := &model.User{ID: 1, Username: "testuser"}
	tokenPair, err := jwtService.GenerateTokenPair(user, types.AuthTypeBasic, nil, nil)
	assert.NoError(t, err)

	mocks.userService.EXPECT().
		GetByID(gomock.Any(), int64(1)).
		Return(&model.User{
			ID:       1,
			Username: "testuser",
			Active:   types.Ptr(true),
		}, nil)

	mocks.roleService.EXPECT().
		GetPermissionsByUsername(gomock.Any(), "testuser").
		Return(&model.SubjectPermissions{}, nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)

	var userCtx *UserContext
	handler := middleware(func(c echo.Context) error {
		userCtx = GetUser(c.Request().Context())
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.NotNil(t, userCtx)
	assert.Equal(t, int64(1), userCtx.UserID)
	assert.Equal(t, "testuser", userCtx.Username)
	assert.Equal(t, types.AuthTypeBasic, userCtx.AuthType)
}

func TestUserCtxAuthMiddleware_JWT_Invalid(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Authorization token")
}

func TestUserCtxAuthMiddleware_JWT_WithExtraRoles(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	// Generate a valid JWT token with extra roles
	jwtService := jwt.NewServiceJWT(jwtConfig)
	user := &model.User{ID: 1, Username: "testuser"}
	tokenPair, err := jwtService.GenerateTokenPair(user, types.AuthTypeBasic, nil, []string{"admin", "editor"})
	assert.NoError(t, err)

	mocks.userService.EXPECT().
		GetByID(gomock.Any(), int64(1)).
		Return(&model.User{
			ID:       1,
			Username: "testuser",
			Active:   types.Ptr(true),
		}, nil)

	mocks.roleService.EXPECT().
		GetPermissionsByUsername(gomock.Any(), "testuser").
		Return(&model.SubjectPermissions{}, nil)

	mocks.roleService.EXPECT().
		GetPermissionsByRoleCode(gomock.Any(), "admin").
		Return(&model.SubjectPermissions{
			Admin: []model.AdminPermission{{Section: model.AdminSectionUsers, Action: model.ActionRead}},
		}, nil)

	mocks.roleService.EXPECT().
		GetPermissionsByRoleCode(gomock.Any(), "editor").
		Return(&model.SubjectPermissions{
			Resources: []model.ResourcePermission{{Namespace: "ns1", Action: model.ActionWrite}},
		}, nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)

	var userCtx *UserContext
	handler := middleware(func(c echo.Context) error {
		userCtx = GetUser(c.Request().Context())
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.NoError(t, err)
	assert.NotNil(t, userCtx)
	assert.Len(t, userCtx.SubjectPermissions.Admin, 1)
	assert.Len(t, userCtx.SubjectPermissions.Resources, 1)
}

func TestUserCtxAuthMiddleware_JWT_UserNotFound(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	// Generate a valid JWT token for non-existent user
	jwtService := jwt.NewServiceJWT(jwtConfig)
	user := &model.User{ID: 999, Username: "nonexistent"}
	tokenPair, err := jwtService.GenerateTokenPair(user, types.AuthTypeBasic, nil, nil)
	assert.NoError(t, err)

	mocks.userService.EXPECT().
		GetByID(gomock.Any(), int64(999)).
		Return(nil, service.ErrUserNotFound)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.Error(t, err)
	assert.Equal(t, service.ErrUserNotFound, err)
}

func TestUserCtxAuthMiddleware_JWT_InactiveUser(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	jwtService := jwt.NewServiceJWT(jwtConfig)
	user := &model.User{ID: 1, Username: "inactiveuser"}
	tokenPair, err := jwtService.GenerateTokenPair(user, types.AuthTypeBasic, nil, nil)
	assert.NoError(t, err)

	mocks.userService.EXPECT().
		GetByID(gomock.Any(), int64(1)).
		Return(&model.User{
			ID:       1,
			Username: "inactiveuser",
			Active:   types.Ptr(false),
		}, nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.Error(t, err)
	assert.Equal(t, service.ErrUserNotFound, err)
}

func TestUserCtxAuthMiddleware_JWT_PermissionsError(t *testing.T) {
	mocks, jwtConfig := setupMiddlewareMocks(t)
	defer mocks.ctrl.Finish()

	jwtService := jwt.NewServiceJWT(jwtConfig)
	user := &model.User{ID: 1, Username: "testuser"}
	tokenPair, err := jwtService.GenerateTokenPair(user, types.AuthTypeBasic, nil, nil)
	assert.NoError(t, err)

	expectedErr := errors.New("permissions error")

	mocks.userService.EXPECT().
		GetByID(gomock.Any(), int64(1)).
		Return(&model.User{
			ID:       1,
			Username: "testuser",
			Active:   types.Ptr(true),
		}, nil)

	mocks.roleService.EXPECT().
		GetPermissionsByUsername(gomock.Any(), "testuser").
		Return(nil, expectedErr)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := UserCtxAuthMiddleware(jwtConfig, mocks.userService, mocks.roleService, mocks.tokenService)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestUserContext_GetUserIdStr(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		want   string
	}{
		{
			name:   "positive id",
			userID: 123,
			want:   "123",
		},
		{
			name:   "zero id (API token)",
			userID: 0,
			want:   "0",
		},
		{
			name:   "large id",
			userID: 9999999999,
			want:   "9999999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &UserContext{UserID: tt.userID}
			assert.Equal(t, tt.want, uc.GetUserIdStr())
		})
	}
}

func TestGetUser(t *testing.T) {
	t.Run("with user context", func(t *testing.T) {
		userCtx := &UserContext{
			UserID:   1,
			Username: "testuser",
		}
		ctx := SetUserContext(context.Background(), userCtx)

		result := GetUser(ctx)
		assert.Equal(t, userCtx, result)
	})

	t.Run("without user context", func(t *testing.T) {
		ctx := context.Background()

		result := GetUser(ctx)
		assert.Nil(t, result)
	})
}

func TestSetUserContext(t *testing.T) {
	userCtx := &UserContext{
		UserID:   1,
		Username: "testuser",
		AuthType: types.AuthTypeBasic,
	}

	ctx := SetUserContext(context.Background(), userCtx)

	result := GetUser(ctx)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.UserID)
	assert.Equal(t, "testuser", result.Username)
	assert.Equal(t, types.AuthTypeBasic, result.AuthType)
}
