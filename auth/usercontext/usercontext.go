// Package usercontext carries the authenticated subject through a request context.
//
// It lives outside the auth package so that layers auth itself depends on -
// services, typically - can still read who is performing an action without
// creating an import cycle.
package usercontext

import (
	"context"
	"strconv"

	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
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

// IsUser reports whether the subject is a real user account. API tokens
// authenticate without one and carry the token name instead.
func (uc UserContext) IsUser() bool {
	return uc.UserID != 0
}

func GetUser(ctx context.Context) *UserContext {
	user, _ := ctx.Value(userCtxKey).(*UserContext)
	return user
}

// SetUserContext adds a UserContext to the given context.
func SetUserContext(ctx context.Context, userCtx *UserContext) context.Context {
	return context.WithValue(ctx, userCtxKey, userCtx)
}
