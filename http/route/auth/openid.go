package auth

import (
	"errors"
	"net/http"

	"github.com/flectolab/flecto-manager/auth/openid"
	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/types"
	"github.com/labstack/echo/v4"
)

const stateSessionKey = "openid_state"

type OpenIDConfigResponse struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name,omitempty"`
	Icon    string `json:"icon,omitempty"`
	AuthURL string `json:"authUrl,omitempty"`
}

func GetOpenIDConfig(cfg *config.OpenIDConfig, openidService openid.Service) func(echo.Context) error {
	return func(c echo.Context) error {
		if !cfg.Enabled {
			return c.JSON(http.StatusOK, OpenIDConfigResponse{
				Enabled: false,
			})
		}

		authURL, state, err := openidService.BeginAuth()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to generate auth URL",
			})
		}

		setStateCookie(c, state)

		return c.JSON(http.StatusOK, OpenIDConfigResponse{
			Enabled: true,
			Name:    cfg.Name,
			Icon:    cfg.Icon,
			AuthURL: authURL,
		})
	}
}

func GetOpenIDCallback(openidService openid.Service) func(echo.Context) error {
	return func(c echo.Context) error {
		code := c.QueryParam("code")
		state := c.QueryParam("state")

		if code == "" {
			errorParam := c.QueryParam("error")
			errorDesc := c.QueryParam("error_description")
			if errorParam != "" {
				return c.Redirect(http.StatusTemporaryRedirect,
					"/login?error="+errorParam+"&error_description="+errorDesc)
			}
			return c.Redirect(http.StatusTemporaryRedirect,
				"/login?error=missing_code&error_description=Authorization+code+is+required")
		}

		expectedState := getStateCookie(c)
		clearStateCookie(c)

		_, tokens, err := openidService.CompleteAuth(c.Request().Context(), code, state, expectedState)
		if err != nil {
			var errorCode, errorDesc string
			switch {
			case errors.Is(err, openid.ErrInvalidState):
				errorCode = "invalid_state"
				errorDesc = "Invalid+state+parameter"
			case errors.Is(err, openid.ErrUserInactive):
				errorCode = "user_inactive"
				errorDesc = "User+account+is+inactive"
			default:
				errorCode = "auth_failed"
				errorDesc = "Authentication+failed"
			}
			return c.Redirect(http.StatusTemporaryRedirect,
				"/login?error="+errorCode+"&error_description="+errorDesc)
		}

		// Redirect to frontend callback page with tokens
		return c.Redirect(http.StatusTemporaryRedirect,
			"/login/callback?access_token="+tokens.AccessToken+"&refresh_token="+tokens.RefreshToken)
	}
}

func setStateCookie(c echo.Context, state string) {
	cookie := &http.Cookie{
		Name:     stateSessionKey,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300, // 5 minutes
	}
	c.SetCookie(cookie)
}

func getStateCookie(c echo.Context) string {
	cookie, err := c.Cookie(stateSessionKey)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func clearStateCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     stateSessionKey,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	c.SetCookie(cookie)
}