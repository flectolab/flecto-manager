package auth

import (
	"errors"
	"net/http"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/labstack/echo/v4"
)

func GetLogin(ctx *appContext.Context, authService service.AuthService) func(echo.Context) error {
	return func(c echo.Context) error {
		var req types.LoginRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_request",
				Message: "Invalid request body",
			})
		}

		if err := ctx.Validator.Struct(&req); err != nil {
			return c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		}

		user, tokens, err := authService.Login(c.Request().Context(), &req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidCredentials):
				return c.JSON(http.StatusUnauthorized, types.ErrorResponse{
					Error:   "invalid_credentials",
					Message: "Invalid email or password",
				})
			case errors.Is(err, service.ErrUserNotFound):
				return c.JSON(http.StatusForbidden, types.ErrorResponse{
					Error:   "user_not_exist",
					Message: "User account not exist",
				})
			default:
				return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
					Error:   "internal_error",
					Message: "Authentication failed",
				})
			}
		}

		return c.JSON(http.StatusOK, types.AuthResponse{
			User:   authService.ToUserResponse(user),
			Tokens: tokens,
		})
	}
}
