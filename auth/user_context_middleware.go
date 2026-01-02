package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/flectolab/flecto-manager/config"
	flectoJwt "github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type contextKey string

const userCtxKey contextKey = "user"

type UserContext struct {
	UserID             int64
	Username           string
	SubjectPermissions *model.SubjectPermissions
	AuthType           types.AuthType
}

func (uc UserContext) GetUserIdStr() string {
	return strconv.FormatInt(uc.UserID, 10)
}

func GetUser(ctx context.Context) *UserContext {
	user, _ := ctx.Value(userCtxKey).(*UserContext)
	return user
}

// SetUserContext adds a UserContext to the given context. This is primarily used for testing.
func SetUserContext(ctx context.Context, userCtx *UserContext) context.Context {
	return context.WithValue(ctx, userCtxKey, userCtx)
}

func UserCtxAuthMiddleware(jwtConfig *config.JWTConfig, userService service.UserService, roleService service.RoleService, tokenService service.TokenService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get(jwtConfig.HeaderName)
			if len(authHeader) <= 7 || authHeader[:7] != "Bearer " {
				return errors.New("missing or invalid Authorization header")
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
		return errors.New("invalid API token")
	}

	ctx := context.WithValue(c.Request().Context(), userCtxKey, &UserContext{
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
		return errors.New("invalid Authorization token")
	}

	if claims, ok := token.Claims.(*flectoJwt.Claims); ok && claims.TokenType == types.TokenTypeAccess {
		subjectPermissions := claims.SubjectPermissions
		if subjectPermissions == nil {
			subjectPermissions = &model.SubjectPermissions{}
		}

		user, errGetUser := userService.GetByID(context.Background(), claims.UserID)
		if errGetUser != nil || !*user.Active {
			return service.ErrUserNotFound
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

		ctx := context.WithValue(c.Request().Context(), userCtxKey, &UserContext{
			UserID:             claims.UserID,
			Username:           claims.Username,
			AuthType:           claims.AuthType,
			SubjectPermissions: subjectPermissions,
		})
		c.SetRequest(c.Request().WithContext(ctx))
	}

	return next(c)
}
