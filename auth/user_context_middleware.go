package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/flectolab/flecto-manager/auth/usercontext"
	"github.com/flectolab/flecto-manager/config"
	flectoJwt "github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// UserContext is defined in the usercontext package so that services, which auth
// depends on, can read the current subject without an import cycle.
type UserContext = usercontext.UserContext

func GetUser(ctx context.Context) *UserContext {
	return usercontext.GetUser(ctx)
}

// SetUserContext adds a UserContext to the given context. This is primarily used for testing.
func SetUserContext(ctx context.Context, userCtx *UserContext) context.Context {
	return usercontext.SetUserContext(ctx, userCtx)
}

// errUnauthorized reports an authentication failure as a 401. Returning a bare
// error instead would surface as a 500, leaving a client unable to tell an expired
// token from a broken server.
func errUnauthorized(message string) error {
	return echo.NewHTTPError(http.StatusUnauthorized, message)
}

func UserCtxAuthMiddleware(jwtConfig *config.JWTConfig, userService service.UserService, roleService service.RoleService, tokenService service.TokenService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get(jwtConfig.HeaderName)
			if len(authHeader) <= 7 || authHeader[:7] != "Bearer " {
				return errUnauthorized("missing or invalid Authorization header")
			}

			token := authHeader[7:]

			// API Token auth (prefixed by flecto_)
			if strings.HasPrefix(token, model.TokenPrefix) {
				return handleAPITokenAuth(c, next, tokenService, token)
			}

			// JWT auth (existing)
			return handleJWTAuth(c, next, jwtConfig, userService, roleService, token)
		}
	}
}

func handleAPITokenAuth(c echo.Context, next echo.HandlerFunc, tokenService service.TokenService, plainToken string) error {
	token, permissions, err := tokenService.ValidateToken(context.Background(), plainToken)
	if err != nil {
		return errUnauthorized("invalid API token")
	}

	ctx := usercontext.SetUserContext(c.Request().Context(), &UserContext{
		UserID:             0,
		Username:           token.Name,
		AuthType:           types.AuthTypeToken,
		SubjectPermissions: permissions,
	})
	c.SetRequest(c.Request().WithContext(ctx))

	return next(c)
}

func handleJWTAuth(c echo.Context, next echo.HandlerFunc, jwtConfig *config.JWTConfig, userService service.UserService, roleService service.RoleService, tokenString string) error {
	token, err := jwt.ParseWithClaims(tokenString, &flectoJwt.Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtConfig.Secret), nil
	})
	if err != nil || !token.Valid {
		return errUnauthorized("invalid Authorization token")
	}

	// Anything but an access token is rejected here. Letting a refresh token
	// through would call the handler with no subject in the context, which the
	// handlers dereference without checking.
	claims, ok := token.Claims.(*flectoJwt.Claims)
	if !ok || claims.TokenType != types.TokenTypeAccess {
		return errUnauthorized("an access token is required")
	}

	subjectPermissions := claims.SubjectPermissions
	if subjectPermissions == nil {
		subjectPermissions = &model.SubjectPermissions{}
	}

	user, errGetUser := userService.GetByID(context.Background(), claims.UserID)
	if errGetUser != nil || !user.IsActive() {
		return errUnauthorized(service.ErrUserNotFound.Error())
	}

	userPermissions, errUserPerm := roleService.GetPermissionsByUsername(context.Background(), user.Username)
	if errUserPerm != nil {
		return errUserPerm
	}
	subjectPermissions.Append(userPermissions)

	for _, role := range claims.ExtraRoles {
		rolePermissions, errRolePerm := roleService.GetPermissionsByRoleCode(context.Background(), role)
		if errRolePerm == nil && rolePermissions != nil {
			subjectPermissions.Append(rolePermissions)
		}
	}

	ctx := usercontext.SetUserContext(c.Request().Context(), &UserContext{
		UserID:             claims.UserID,
		Username:           claims.Username,
		AuthType:           claims.AuthType,
		SubjectPermissions: subjectPermissions,
	})
	c.SetRequest(c.Request().WithContext(ctx))

	return next(c)
}
