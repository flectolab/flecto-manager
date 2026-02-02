package auth

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flectolab/flecto-manager/auth/openid"
	"github.com/flectolab/flecto-manager/config"
	appContext "github.com/flectolab/flecto-manager/context"
	mockOpenID "github.com/flectolab/flecto-manager/mocks/flecto-manager/auth/openid"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetOpenIDConfig(t *testing.T) {
	t.Run("disabled returns enabled false", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		cfg := &config.OpenIDConfig{
			Enabled: false,
		}
		mockService := mockOpenID.NewMockService(ctrl)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDConfig(ctx, cfg, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"enabled":false`)
	})

	t.Run("enabled returns auth URL and sets cookie", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		cfg := &config.OpenIDConfig{
			Enabled: true,
			Name:    "Google",
			Icon:    "google",
		}
		mockService := mockOpenID.NewMockService(ctrl)

		mockService.EXPECT().
			BeginAuth().
			Return("https://provider.com/auth?state=abc123", "abc123", nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDConfig(ctx, cfg, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"enabled":true`)
		assert.Contains(t, rec.Body.String(), `"name":"Google"`)
		assert.Contains(t, rec.Body.String(), `"icon":"google"`)
		assert.Contains(t, rec.Body.String(), `"authUrl":"https://provider.com/auth?state=abc123"`)

		// Check cookie is set
		cookies := rec.Result().Cookies()
		var stateCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == stateSessionKey {
				stateCookie = cookie
				break
			}
		}
		require.NotNil(t, stateCookie)
		assert.Equal(t, "abc123", stateCookie.Value)
		assert.True(t, stateCookie.HttpOnly)
	})

	t.Run("error on BeginAuth returns internal error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		cfg := &config.OpenIDConfig{
			Enabled: true,
		}
		mockService := mockOpenID.NewMockService(ctrl)

		mockService.EXPECT().
			BeginAuth().
			Return("", "", errors.New("failed to generate state"))

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDConfig(ctx, cfg, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), `"error":"internal_error"`)
	})
}

func TestGetOpenIDCallback(t *testing.T) {
	t.Run("success redirects with tokens", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockService := mockOpenID.NewMockService(ctrl)

		user := &model.User{ID: 1, Username: "test@example.com"}
		tokens := &types.TokenPair{
			AccessToken:  "access-token-123",
			RefreshToken: "refresh-token-456",
		}

		mockService.EXPECT().
			CompleteAuth(gomock.Any(), "auth-code", "state-123", "state-123").
			Return(user, tokens, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid/callback?code=auth-code&state=state-123", nil)
		req.AddCookie(&http.Cookie{Name: stateSessionKey, Value: "state-123"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDCallback(ctx, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "/login/callback?access_token=access-token-123&refresh_token=refresh-token-456")
	})

	t.Run("missing code with error param redirects with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockService := mockOpenID.NewMockService(ctrl)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid/callback?error=access_denied&error_description=User+denied+access", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDCallback(ctx, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		location := rec.Header().Get("Location")
		assert.Contains(t, location, "/login?error=access_denied")
		assert.Contains(t, location, "error_description=User")
	})

	t.Run("missing code without error param redirects with missing_code error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockService := mockOpenID.NewMockService(ctrl)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid/callback", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDCallback(ctx, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "/login?error=missing_code")
	})

	t.Run("invalid state redirects with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockService := mockOpenID.NewMockService(ctrl)

		mockService.EXPECT().
			CompleteAuth(gomock.Any(), "auth-code", "wrong-state", "expected-state").
			Return(nil, nil, openid.ErrInvalidState)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid/callback?code=auth-code&state=wrong-state", nil)
		req.AddCookie(&http.Cookie{Name: stateSessionKey, Value: "expected-state"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDCallback(ctx, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "/login?error=invalid_state")
	})

	t.Run("user inactive redirects with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockService := mockOpenID.NewMockService(ctrl)

		mockService.EXPECT().
			CompleteAuth(gomock.Any(), "auth-code", "state-123", "state-123").
			Return(nil, nil, openid.ErrUserInactive)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid/callback?code=auth-code&state=state-123", nil)
		req.AddCookie(&http.Cookie{Name: stateSessionKey, Value: "state-123"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDCallback(ctx, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "/login?error=user_inactive")
	})

	t.Run("generic auth error redirects with auth_failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockService := mockOpenID.NewMockService(ctrl)

		mockService.EXPECT().
			CompleteAuth(gomock.Any(), "auth-code", "state-123", "state-123").
			Return(nil, nil, errors.New("some internal error"))

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid/callback?code=auth-code&state=state-123", nil)
		req.AddCookie(&http.Cookie{Name: stateSessionKey, Value: "state-123"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDCallback(ctx, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "/login?error=auth_failed")
	})

	t.Run("no state cookie returns empty expected state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := appContext.TestContext(nil)
		mockService := mockOpenID.NewMockService(ctrl)

		mockService.EXPECT().
			CompleteAuth(gomock.Any(), "auth-code", "state-123", "").
			Return(nil, nil, openid.ErrInvalidState)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/openid/callback?code=auth-code&state=state-123", nil)
		// No cookie set
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := GetOpenIDCallback(ctx, mockService)
		err := handler(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "/login?error=invalid_state")
	})
}

func TestStateCookieFunctions(t *testing.T) {
	t.Run("setStateCookie sets cookie with correct attributes", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setStateCookie(c, "test-state-value")

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)

		cookie := cookies[0]
		assert.Equal(t, stateSessionKey, cookie.Name)
		assert.Equal(t, "test-state-value", cookie.Value)
		assert.Equal(t, "/", cookie.Path)
		assert.True(t, cookie.HttpOnly)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.Equal(t, 300, cookie.MaxAge)
	})

	t.Run("setStateCookie with TLS sets Secure flag", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
		req.TLS = &tls.ConnectionState{} // Simulate TLS connection
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setStateCookie(c, "test-state-value")

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.True(t, cookies[0].Secure)
	})

	t.Run("getStateCookie returns value when cookie exists", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: stateSessionKey, Value: "my-state"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		value := getStateCookie(c)

		assert.Equal(t, "my-state", value)
	})

	t.Run("getStateCookie returns empty string when cookie not exists", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		value := getStateCookie(c)

		assert.Equal(t, "", value)
	})

	t.Run("clearStateCookie sets cookie with MaxAge -1", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		clearStateCookie(c)

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)

		cookie := cookies[0]
		assert.Equal(t, stateSessionKey, cookie.Name)
		assert.Equal(t, "", cookie.Value)
		assert.Equal(t, -1, cookie.MaxAge)
	})
}