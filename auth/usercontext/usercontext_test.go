package usercontext

import (
	"context"
	"testing"

	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
)

func TestGetUser(t *testing.T) {
	t.Run("returns the subject set on the context", func(t *testing.T) {
		expected := &UserContext{
			UserID:             42,
			Username:           "alice",
			AuthType:           types.AuthTypeBasic,
			SubjectPermissions: &model.SubjectPermissions{},
		}

		ctx := SetUserContext(context.Background(), expected)

		assert.Same(t, expected, GetUser(ctx))
	})

	t.Run("returns nil when the context has no subject", func(t *testing.T) {
		assert.Nil(t, GetUser(context.Background()))
	})

	t.Run("returns nil when the context value has another type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userCtxKey, "not-a-user")

		assert.Nil(t, GetUser(ctx))
	})
}

func TestUserContext_GetUserIdStr(t *testing.T) {
	tests := []struct {
		name    string
		subject UserContext
		want    string
	}{
		{name: "user", subject: UserContext{UserID: 42}, want: "42"},
		{name: "api token", subject: UserContext{UserID: 0}, want: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.subject.GetUserIdStr())
		})
	}
}

func TestUserContext_IsUser(t *testing.T) {
	tests := []struct {
		name    string
		subject UserContext
		want    bool
	}{
		{name: "user account", subject: UserContext{UserID: 42, Username: "alice"}, want: true},
		// API tokens authenticate without a user account and report id 0.
		{name: "api token", subject: UserContext{UserID: 0, Username: "ci-token", AuthType: types.AuthTypeToken}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.subject.IsUser())
		})
	}
}
