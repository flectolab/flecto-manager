package auth

import (
	"errors"
	"net/http"

	"github.com/flectolab/flecto-manager/config"
	flectoJwt "github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

const (
	// ContextKeyUser is the key used to store user claims in echo context
	ContextKeyUser = "user"
)

type Handler struct {
	authService *AuthService
	jwtService  *flectoJwt.ServiceJWT
	jwtConfig   *config.JWTConfig
	validate    *validator.Validate
}

func NewHandler(authService *AuthService, jwtService *flectoJwt.ServiceJWT, jwtConfig *config.JWTConfig) *Handler {
	return &Handler{
		authService: authService,
		jwtService:  jwtService,
		jwtConfig:   jwtConfig,
		validate:    validator.New(),
	}
}

type AuthResponse struct {
	User   *UserResponse        `json:"user"`
	Tokens *flectoJwt.TokenPair `json:"tokens"`
}

type UserResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Login handles user authentication
func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
	}

	user, tokens, err := h.authService.Login(c.Request().Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			return c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_credentials",
				Message: "Invalid email or password",
			})
		case errors.Is(err, ErrUserInactive):
			return c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "user_inactive",
				Message: "User account is inactive",
			})
		case errors.Is(err, ErrNoPassword):
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "no_password",
				Message: "User has no password set, use OpenID login",
			})
		default:
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Authentication failed",
			})
		}
	}

	return c.JSON(http.StatusOK, AuthResponse{
		User:   toUserResponse(user),
		Tokens: tokens,
	})
}

// Refresh handles token refresh
func (h *Handler) Refresh(c echo.Context) error {
	var req RefreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
	}

	// Parse and validate refresh token
	token, err := jwt.ParseWithClaims(req.RefreshToken, &flectoJwt.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return h.jwtService.GetSecret(), nil
	})
	if err != nil {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_token",
			Message: "Invalid or expired refresh token",
		})
	}

	claims, ok := token.Claims.(*flectoJwt.Claims)
	if !ok || !token.Valid {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_token",
			Message: "Invalid refresh token",
		})
	}

	user, tokens, err := h.authService.RefreshTokens(c.Request().Context(), req.RefreshToken, claims)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			return c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_token",
				Message: "Refresh token has been revoked",
			})
		case errors.Is(err, ErrUserInactive):
			return c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "user_inactive",
				Message: "User account is inactive",
			})
		default:
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Token refresh failed",
			})
		}
	}

	return c.JSON(http.StatusOK, AuthResponse{
		User:   toUserResponse(user),
		Tokens: tokens,
	})
}

func GetUserClaims(c echo.Context) *flectoJwt.Claims {
	token := c.Get(ContextKeyUser).(*jwt.Token)
	return token.Claims.(*flectoJwt.Claims)
}

// Logout handles user logout (invalidates refresh token)
func (h *Handler) Logout(c echo.Context) error {
	claims := GetUserClaims(c)

	if err := h.authService.Logout(c.Request().Context(), claims.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Logout failed",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// RegisterRoutes registers all auth routes on the Echo instance
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	authGroup := e.Group("/auth")

	// Public routes
	authGroup.POST("/login", h.Login)
	authGroup.POST("/refresh", h.Refresh)

	// Protected routes
	protectedGroup := authGroup.Group("")
	protectedGroup.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(h.jwtConfig.Secret),
		ContextKey:  ContextKeyUser,
		TokenLookup: "header:" + h.jwtConfig.HeaderName + ":Bearer ",
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return &flectoJwt.Claims{}
		},
	}))
	protectedGroup.POST("/logout", h.Logout)
}

func toUserResponse(user *model.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
	}
}
