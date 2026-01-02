package openid_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/flectolab/flecto-manager/auth/openid"
	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/jwt"
	mockOpenID "github.com/flectolab/flecto-manager/mocks/flecto-manager/auth/openid"
	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
)

func setupServiceTest(t *testing.T) (
	*gomock.Controller,
	*mockOpenID.MockProvider,
	*mockFlectoService.MockUserService,
	*jwt.ServiceJWT,
	openid.Service,
) {
	ctrl := gomock.NewController(t)
	mockProvider := mockOpenID.NewMockProvider(ctrl)
	mockUserService := mockFlectoService.NewMockUserService(ctrl)
	jwtService := jwt.NewServiceJWT(&config.JWTConfig{
		Secret:          "test-secret-key-32-bytes-long!!!",
		Issuer:          "test-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})
	svc := openid.NewService(mockProvider, mockUserService, jwtService)
	return ctrl, mockProvider, mockUserService, jwtService, svc
}

func TestNewService(t *testing.T) {
	ctrl, mockProvider, mockUserService, jwtService, svc := setupServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockProvider)
	assert.NotNil(t, mockUserService)
	assert.NotNil(t, jwtService)
}

func TestService_BeginAuth(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockProvider, _, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		expectedAuthURL := "https://provider.com/auth?state="

		mockProvider.EXPECT().
			GetAuthURL(gomock.Any()).
			DoAndReturn(func(state string) string {
				assert.NotEmpty(t, state)
				assert.Len(t, state, 44) // base64 encoded 32 bytes
				return expectedAuthURL + state
			})

		authURL, state, err := svc.BeginAuth()

		assert.NoError(t, err)
		assert.NotEmpty(t, authURL)
		assert.NotEmpty(t, state)
		assert.Contains(t, authURL, expectedAuthURL)
	})
}

func TestService_CompleteAuth(t *testing.T) {
	t.Run("invalid state", func(t *testing.T) {
		ctrl, _, _, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		user, tokens, err := svc.CompleteAuth(ctx, "code", "state1", "state2")

		assert.Equal(t, openid.ErrInvalidState, err)
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})

	t.Run("exchange error", func(t *testing.T) {
		ctrl, mockProvider, _, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		mockProvider.EXPECT().
			Exchange(ctx, "invalid-code").
			Return(nil, errors.New("exchange failed"))

		user, tokens, err := svc.CompleteAuth(ctx, "invalid-code", state, state)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to exchange code")
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})

	t.Run("no id_token in response", func(t *testing.T) {
		ctrl, mockProvider, _, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		// Token without id_token extra
		token := &oauth2.Token{
			AccessToken: "access-token",
		}

		mockProvider.EXPECT().
			Exchange(ctx, "code").
			Return(token, nil)

		user, tokens, err := svc.CompleteAuth(ctx, "code", state, state)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no id_token in response")
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})

	t.Run("verify id_token error", func(t *testing.T) {
		ctrl, mockProvider, _, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		// Token with id_token extra
		token := (&oauth2.Token{
			AccessToken: "access-token",
		}).WithExtra(map[string]interface{}{
			"id_token": "raw-id-token",
		})

		mockProvider.EXPECT().
			Exchange(ctx, "code").
			Return(token, nil)

		mockProvider.EXPECT().
			VerifyIDToken(ctx, "raw-id-token").
			Return(nil, errors.New("invalid token"))

		user, tokens, err := svc.CompleteAuth(ctx, "code", state, state)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify ID token")
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})

	t.Run("get user info error", func(t *testing.T) {
		ctrl, mockProvider, _, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		token := (&oauth2.Token{
			AccessToken: "access-token",
		}).WithExtra(map[string]interface{}{
			"id_token": "raw-id-token",
		})

		idToken := &oidc.IDToken{}

		mockProvider.EXPECT().
			Exchange(ctx, "code").
			Return(token, nil)

		mockProvider.EXPECT().
			VerifyIDToken(ctx, "raw-id-token").
			Return(idToken, nil)

		mockProvider.EXPECT().
			GetUserInfo(ctx, token, idToken).
			Return(nil, errors.New("failed to get user info"))

		user, tokens, err := svc.CompleteAuth(ctx, "code", state, state)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get user info")
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})

	t.Run("find or create user error", func(t *testing.T) {
		ctrl, mockProvider, mockUserService, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		token := (&oauth2.Token{
			AccessToken: "access-token",
		}).WithExtra(map[string]interface{}{
			"id_token": "raw-id-token",
		})

		idToken := &oidc.IDToken{}

		userInfo := &openid.UserInfo{
			Subject:   "subject-123",
			Email:     "user@example.com",
			FirstName: "John",
			LastName:  "Doe",
		}

		mockProvider.EXPECT().
			Exchange(ctx, "code").
			Return(token, nil)

		mockProvider.EXPECT().
			VerifyIDToken(ctx, "raw-id-token").
			Return(idToken, nil)

		mockProvider.EXPECT().
			GetUserInfo(ctx, token, idToken).
			Return(userInfo, nil)

		mockUserService.EXPECT().
			FindOrCreate(ctx, gomock.Any()).
			Return(nil, errors.New("database error"))

		user, tokens, err := svc.CompleteAuth(ctx, "code", state, state)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find or create user")
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})

	t.Run("user inactive", func(t *testing.T) {
		ctrl, mockProvider, mockUserService, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		token := (&oauth2.Token{
			AccessToken: "access-token",
		}).WithExtra(map[string]interface{}{
			"id_token": "raw-id-token",
		})

		idToken := &oidc.IDToken{}

		userInfo := &openid.UserInfo{
			Subject:   "subject-123",
			Email:     "user@example.com",
			FirstName: "John",
			LastName:  "Doe",
		}

		inactive := false
		existingUser := &model.User{
			ID:       1,
			Username: "user@example.com",
			Active:   &inactive,
		}

		mockProvider.EXPECT().
			Exchange(ctx, "code").
			Return(token, nil)

		mockProvider.EXPECT().
			VerifyIDToken(ctx, "raw-id-token").
			Return(idToken, nil)

		mockProvider.EXPECT().
			GetUserInfo(ctx, token, idToken).
			Return(userInfo, nil)

		mockUserService.EXPECT().
			FindOrCreate(ctx, gomock.Any()).
			Return(existingUser, nil)

		user, tokens, err := svc.CompleteAuth(ctx, "code", state, state)

		assert.Equal(t, openid.ErrUserInactive, err)
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})

	t.Run("success with existing user", func(t *testing.T) {
		ctrl, mockProvider, mockUserService, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		token := (&oauth2.Token{
			AccessToken: "access-token",
		}).WithExtra(map[string]interface{}{
			"id_token": "raw-id-token",
		})

		idToken := &oidc.IDToken{}

		userInfo := &openid.UserInfo{
			Subject:   "subject-123",
			Email:     "user@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Roles:     []string{"admin", "user"},
		}

		active := true
		existingUser := &model.User{
			ID:        1,
			Username:  "user@example.com",
			Firstname: "John",
			Lastname:  "Doe",
			Active:    &active,
		}

		mockProvider.EXPECT().
			Exchange(ctx, "code").
			Return(token, nil)

		mockProvider.EXPECT().
			VerifyIDToken(ctx, "raw-id-token").
			Return(idToken, nil)

		mockProvider.EXPECT().
			GetUserInfo(ctx, token, idToken).
			Return(userInfo, nil)

		mockUserService.EXPECT().
			FindOrCreate(ctx, gomock.Any()).
			Return(existingUser, nil)

		mockUserService.EXPECT().
			UpdateRefreshToken(ctx, int64(1), gomock.Any()).
			Return(nil)

		user, tokens, err := svc.CompleteAuth(ctx, "code", state, state)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "user@example.com", user.Username)
		assert.NotNil(t, tokens)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("success with new user creation", func(t *testing.T) {
		ctrl, mockProvider, mockUserService, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		token := (&oauth2.Token{
			AccessToken: "access-token",
		}).WithExtra(map[string]interface{}{
			"id_token": "raw-id-token",
		})

		idToken := &oidc.IDToken{}

		userInfo := &openid.UserInfo{
			Subject:   "subject-123",
			Email:     "newuser@example.com",
			FirstName: "Jane",
			LastName:  "Smith",
		}

		active := true
		newUser := &model.User{
			ID:        1,
			Username:  "newuser@example.com",
			Firstname: "Jane",
			Lastname:  "Smith",
			Active:    &active,
		}

		mockProvider.EXPECT().
			Exchange(ctx, "code").
			Return(token, nil)

		mockProvider.EXPECT().
			VerifyIDToken(ctx, "raw-id-token").
			Return(idToken, nil)

		mockProvider.EXPECT().
			GetUserInfo(ctx, token, idToken).
			Return(userInfo, nil)

		mockUserService.EXPECT().
			FindOrCreate(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, u *model.User) (*model.User, error) {
				assert.Equal(t, "newuser@example.com", u.Username)
				assert.Equal(t, "Jane", u.Firstname)
				assert.Equal(t, "Smith", u.Lastname)
				assert.True(t, *u.Active)
				return newUser, nil
			})

		mockUserService.EXPECT().
			UpdateRefreshToken(ctx, int64(1), gomock.Any()).
			Return(nil)

		user, tokens, err := svc.CompleteAuth(ctx, "code", state, state)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "newuser@example.com", user.Username)
		assert.NotNil(t, tokens)
	})

	t.Run("success with subject as username when no email", func(t *testing.T) {
		ctrl, mockProvider, mockUserService, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		token := (&oauth2.Token{
			AccessToken: "access-token",
		}).WithExtra(map[string]interface{}{
			"id_token": "raw-id-token",
		})

		idToken := &oidc.IDToken{}

		userInfo := &openid.UserInfo{
			Subject:   "subject-123",
			Email:     "", // No email
			FirstName: "John",
			LastName:  "Doe",
		}

		active := true
		existingUser := &model.User{
			ID:       1,
			Username: "subject-123",
			Active:   &active,
		}

		mockProvider.EXPECT().
			Exchange(ctx, "code").
			Return(token, nil)

		mockProvider.EXPECT().
			VerifyIDToken(ctx, "raw-id-token").
			Return(idToken, nil)

		mockProvider.EXPECT().
			GetUserInfo(ctx, token, idToken).
			Return(userInfo, nil)

		mockUserService.EXPECT().
			FindOrCreate(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, u *model.User) (*model.User, error) {
				assert.Equal(t, "subject-123", u.Username)
				return existingUser, nil
			})

		mockUserService.EXPECT().
			UpdateRefreshToken(ctx, int64(1), gomock.Any()).
			Return(nil)

		user, tokens, err := svc.CompleteAuth(ctx, "code", state, state)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "subject-123", user.Username)
		assert.NotNil(t, tokens)
	})

	t.Run("update refresh token error", func(t *testing.T) {
		ctrl, mockProvider, mockUserService, _, svc := setupServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		state := "valid-state"

		token := (&oauth2.Token{
			AccessToken: "access-token",
		}).WithExtra(map[string]interface{}{
			"id_token": "raw-id-token",
		})

		idToken := &oidc.IDToken{}

		userInfo := &openid.UserInfo{
			Subject:   "subject-123",
			Email:     "user@example.com",
			FirstName: "John",
			LastName:  "Doe",
		}

		active := true
		existingUser := &model.User{
			ID:       1,
			Username: "user@example.com",
			Active:   &active,
		}

		mockProvider.EXPECT().
			Exchange(ctx, "code").
			Return(token, nil)

		mockProvider.EXPECT().
			VerifyIDToken(ctx, "raw-id-token").
			Return(idToken, nil)

		mockProvider.EXPECT().
			GetUserInfo(ctx, token, idToken).
			Return(userInfo, nil)

		mockUserService.EXPECT().
			FindOrCreate(ctx, gomock.Any()).
			Return(existingUser, nil)

		mockUserService.EXPECT().
			UpdateRefreshToken(ctx, int64(1), gomock.Any()).
			Return(errors.New("database error"))

		user, tokens, err := svc.CompleteAuth(ctx, "code", state, state)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update user")
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})
}

func TestToUserResponse(t *testing.T) {
	tests := []struct {
		name string
		user *model.User
		want *types.UserResponse
	}{
		{
			name: "full user",
			user: &model.User{
				ID:        1,
				Username:  "john@example.com",
				Firstname: "John",
				Lastname:  "Doe",
			},
			want: &types.UserResponse{
				ID:        1,
				Username:  "john@example.com",
				Firstname: "John",
				Lastname:  "Doe",
			},
		},
		{
			name: "user with empty names",
			user: &model.User{
				ID:       2,
				Username: "user123",
			},
			want: &types.UserResponse{
				ID:       2,
				Username: "user123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := openid.ToUserResponse(tt.user)
			assert.Equal(t, tt.want, got)
		})
	}
}
