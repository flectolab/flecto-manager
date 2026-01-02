package auth

import (
	"context"
	"strconv"

	"github.com/flectolab/flecto-manager/config"
	flectoJwt "github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type contextKey string

const userCtxKey contextKey = "user"

type UserContext struct {
	UserID             int64
	Username           string
	SubjectPermissions *model.SubjectPermissions
	AuthType           flectoJwt.AuthType
}

func (uc UserContext) GetUserIdStr() string {
	return strconv.FormatInt(uc.UserID, 10)
}

func GetUser(ctx context.Context) *UserContext {
	user, _ := ctx.Value(userCtxKey).(*UserContext)
	return user
}

func GraphQLAuthMiddleware(jwtConfig *config.JWTConfig, userService service.UserService, roleService service.RoleService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get(jwtConfig.HeaderName)
			if len(authHeader) <= 7 || authHeader[:7] != "Bearer " {
				return next(c)
			}

			token, err := jwt.ParseWithClaims(authHeader[7:], &flectoJwt.Claims{}, func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtConfig.Secret), nil
			})
			if err != nil || !token.Valid {
				return next(c)
			}

			if claims, ok := token.Claims.(*flectoJwt.Claims); ok && claims.TokenType == flectoJwt.TokenTypeAccess {

				subjectPermissions := claims.SubjectPermissions
				if subjectPermissions == nil {
					subjectPermissions = &model.SubjectPermissions{}
				}
				if claims.AuthType != flectoJwt.AuthTypeToken {
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
	}
}
