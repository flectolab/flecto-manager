package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	flectoAuth "github.com/flectolab/flecto-manager/auth"
	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetLogout(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		mockAuthService.EXPECT().
			Logout(gomock.Any(), int64(1)).
			Return(nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)

		// Add user context using the auth package's SetUserContext function
		userCtx := &flectoAuth.UserContext{UserID: 1, Username: "test@example.com"}
		reqCtx := flectoAuth.SetUserContext(req.Context(), userCtx)
		req = req.WithContext(reqCtx)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetLogout(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("logout error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockAuthService := mockFlectoService.NewMockAuthService(ctrl)

		mockAuthService.EXPECT().
			Logout(gomock.Any(), int64(1)).
			Return(errors.New("database error"))

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)

		// Add user context using the auth package's SetUserContext function
		userCtx := &flectoAuth.UserContext{UserID: 1, Username: "test@example.com"}
		reqCtx := flectoAuth.SetUserContext(req.Context(), userCtx)
		req = req.WithContext(reqCtx)

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetLogout(ctx, mockAuthService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"internal_error"`)
	})
}
