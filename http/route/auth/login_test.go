package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		user := &model.User{ID: 1, Username: "test@example.com", Firstname: "John", Lastname: "Doe"}
		tokens := &types.TokenPair{AccessToken: "access-token", RefreshToken: "refresh-token"}
		userResponse := &types.UserResponse{ID: 1, Username: "test@example.com", Firstname: "John", Lastname: "Doe"}

		mockAuthService.EXPECT().
			Login(gomock.Any(), &types.LoginRequest{Username: "test@example.com", Password: "password123"}).
			Return(user, tokens, nil)

		mockAuthService.EXPECT().
			ToUserResponse(user).
			Return(userResponse)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"test@example.com","password":"password123"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetLogin(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"username":"test@example.com"`)
		assert.Contains(t, rec.Body.String(), `"accessToken":"access-token"`)
	})

	t.Run("invalid request body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`invalid json`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetLogin(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"invalid_request"`)
	})

	t.Run("validation error - missing username", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"password":"password123"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetLogin(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"validation_error"`)
	})

	t.Run("validation error - missing password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"test@example.com"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetLogin(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"validation_error"`)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		mockAuthService.EXPECT().
			Login(gomock.Any(), gomock.Any()).
			Return(nil, nil, service.ErrInvalidCredentials)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"test@example.com","password":"wrongpassword"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetLogin(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"invalid_credentials"`)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		mockAuthService.EXPECT().
			Login(gomock.Any(), gomock.Any()).
			Return(nil, nil, service.ErrUserNotFound)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"nonexistent@example.com","password":"password123"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetLogin(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"user_not_exist"`)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		mockAuthService.EXPECT().
			Login(gomock.Any(), gomock.Any()).
			Return(nil, nil, errors.New("database error"))

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"test@example.com","password":"password123"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetLogin(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"internal_error"`)
	})
}
