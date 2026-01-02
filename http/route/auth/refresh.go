package auth

import (
	"errors"
	"net/http"

	appContext "github.com/flectolab/flecto-manager/context"
	flectoJwt "github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func GetRefresh(ctx *appContext.Context, authService service.AuthService) func(echo.Context) error {
	return func(c echo.Context) error {
		var req types.RefreshRequest
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

		// Parse and validate refresh token
		token, err := jwt.ParseWithClaims(req.RefreshToken, &flectoJwt.Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(ctx.Config.Auth.JWT.Secret), nil
		})
		if err != nil {
			return c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired refresh token",
			})
		}

		claims, ok := token.Claims.(*flectoJwt.Claims)
		if !ok || !token.Valid {
			return c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid refresh token",
			})
		}

		user, tokens, err := authService.RefreshTokens(c.Request().Context(), req.RefreshToken, claims)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidCredentials):
				return c.JSON(http.StatusUnauthorized, types.ErrorResponse{
					Error:   "invalid_token",
					Message: "Refresh token has been revoked",
				})
			case errors.Is(err, service.ErrUserInactive):
				return c.JSON(http.StatusForbidden, types.ErrorResponse{
					Error:   "user_inactive",
					Message: "User account is inactive",
				})
			default:
				return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
					Error:   "internal_error",
					Message: "Token refresh failed",
				})
			}
		}

		return c.JSON(http.StatusOK, types.AuthResponse{
			User:   authService.ToUserResponse(user),
			Tokens: tokens,
		})
	}
}
