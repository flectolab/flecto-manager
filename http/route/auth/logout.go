package auth

import (
	"net/http"

	"github.com/flectolab/flecto-manager/auth"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/labstack/echo/v4"
)

func GetLogout(ctx *appContext.Context, authService service.AuthService) func(echo.Context) error {
	return func(c echo.Context) error {
		userCtx := auth.GetUser(c.Request().Context())

		if err := authService.Logout(c.Request().Context(), userCtx.UserID); err != nil {
			return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "internal_error",
				Message: "Logout failed",
			})
		}

		return c.NoContent(http.StatusNoContent)
	}
}
