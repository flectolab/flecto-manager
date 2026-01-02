package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/config"
	appContext "github.com/flectolab/flecto-manager/context"
	flectoJwt "github.com/flectolab/flecto-manager/jwt"
	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createTestRefreshToken(t *testing.T, secret string, userID int64, username string, expired bool) string {
	now := time.Now()
	exp := now.Add(24 * time.Hour)
	if expired {
		exp = now.Add(-1 * time.Hour)
	}

	claims := &flectoJwt.Claims{
		UserID:    userID,
		Username:  username,
		TokenType: types.TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "test-issuer",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func TestGetRefresh(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!!"

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		ctx.Config.Auth.JWT.Secret = secret
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		refreshToken := createTestRefreshToken(t, secret, 1, "test@example.com", false)

		user := &model.User{ID: 1, Username: "test@example.com", Firstname: "John", Lastname: "Doe"}
		tokens := &types.TokenPair{AccessToken: "new-access-token", RefreshToken: "new-refresh-token"}
		userResponse := &types.UserResponse{ID: 1, Username: "test@example.com", Firstname: "John", Lastname: "Doe"}

		mockAuthService.EXPECT().
			RefreshTokens(gomock.Any(), refreshToken, gomock.Any()).
			Return(user, tokens, nil)

		mockAuthService.EXPECT().
			ToUserResponse(user).
			Return(userResponse)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refreshToken":"`+refreshToken+`"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"username":"test@example.com"`)
		assert.Contains(t, rec.Body.String(), `"accessToken":"new-access-token"`)
	})

	t.Run("invalid request body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`invalid json`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"invalid_request"`)
	})

	t.Run("validation error - missing refresh token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"validation_error"`)
	})

	t.Run("invalid token format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		ctx.Config.Auth.JWT.Secret = secret
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refreshToken":"invalid-token"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"invalid_token"`)
		assert.Contains(t, rec.Body.String(), "Invalid or expired refresh token")
	})

	t.Run("expired token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		ctx.Config.Auth.JWT.Secret = secret
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		expiredToken := createTestRefreshToken(t, secret, 1, "test@example.com", true)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refreshToken":"`+expiredToken+`"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"invalid_token"`)
	})

	t.Run("wrong secret", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		ctx.Config.Auth.JWT.Secret = "different-secret-key-32-bytes!!"
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		tokenWithOtherSecret := createTestRefreshToken(t, secret, 1, "test@example.com", false)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refreshToken":"`+tokenWithOtherSecret+`"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"invalid_token"`)
	})

	t.Run("token revoked (invalid credentials)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		ctx.Config.Auth.JWT.Secret = secret
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		refreshToken := createTestRefreshToken(t, secret, 1, "test@example.com", false)

		mockAuthService.EXPECT().
			RefreshTokens(gomock.Any(), refreshToken, gomock.Any()).
			Return(nil, nil, service.ErrInvalidCredentials)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refreshToken":"`+refreshToken+`"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"invalid_token"`)
		assert.Contains(t, rec.Body.String(), "Refresh token has been revoked")
	})

	t.Run("user inactive", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		ctx.Config.Auth.JWT.Secret = secret
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		refreshToken := createTestRefreshToken(t, secret, 1, "test@example.com", false)

		mockAuthService.EXPECT().
			RefreshTokens(gomock.Any(), refreshToken, gomock.Any()).
			Return(nil, nil, service.ErrUserInactive)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refreshToken":"`+refreshToken+`"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"user_inactive"`)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		ctx.Config.Auth.JWT.Secret = secret
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		refreshToken := createTestRefreshToken(t, secret, 1, "test@example.com", false)

		mockAuthService.EXPECT().
			RefreshTokens(gomock.Any(), refreshToken, gomock.Any()).
			Return(nil, nil, errors.New("database error"))

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refreshToken":"`+refreshToken+`"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"internal_error"`)
	})
}


func TestGetRefresh_WithDefaultConfig(t *testing.T) {
	t.Run("uses default config secret", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		// Use the default config which has a default secret
		defaultSecret := config.DefaultConfig().Auth.JWT.Secret
		ctx.Config.Auth.JWT.Secret = defaultSecret
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		refreshToken := createTestRefreshToken(t, defaultSecret, 1, "test@example.com", false)

		user := &model.User{ID: 1, Username: "test@example.com"}
		tokens := &types.TokenPair{AccessToken: "access", RefreshToken: "refresh"}
		userResponse := &types.UserResponse{ID: 1, Username: "test@example.com"}

		mockAuthService.EXPECT().
			RefreshTokens(gomock.Any(), refreshToken, gomock.Any()).
			Return(user, tokens, nil)

		mockAuthService.EXPECT().
			ToUserResponse(user).
			Return(userResponse)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refreshToken":"`+refreshToken+`"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetRefresh(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
