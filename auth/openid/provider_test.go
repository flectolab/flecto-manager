package openid

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRoles(t *testing.T) {
	tests := []struct {
		name   string
		claims map[string]interface{}
		path   string
		want   []string
	}{
		{
			name:   "empty path returns nil",
			claims: map[string]interface{}{"roles": []interface{}{"admin"}},
			path:   "",
			want:   nil,
		},
		{
			name:   "simple path with []interface{}",
			claims: map[string]interface{}{"roles": []interface{}{"admin", "user"}},
			path:   "roles",
			want:   []string{"admin", "user"},
		},
		{
			name:   "simple path with []string",
			claims: map[string]interface{}{"roles": []string{"admin", "user"}},
			path:   "roles",
			want:   []string{"admin", "user"},
		},
		{
			name: "nested path - keycloak realm_access.roles",
			claims: map[string]interface{}{
				"realm_access": map[string]interface{}{
					"roles": []interface{}{"offline_access", "uma_authorization", "admin"},
				},
			},
			path: "realm_access.roles",
			want: []string{"offline_access", "uma_authorization", "admin"},
		},
		{
			name: "deeply nested path",
			claims: map[string]interface{}{
				"resource_access": map[string]interface{}{
					"my-client": map[string]interface{}{
						"roles": []interface{}{"client-admin"},
					},
				},
			},
			path: "resource_access.my-client.roles",
			want: []string{"client-admin"},
		},
		{
			name:   "path not found returns nil",
			claims: map[string]interface{}{"other": "value"},
			path:   "roles",
			want:   nil,
		},
		{
			name: "intermediate path not found returns nil",
			claims: map[string]interface{}{
				"realm_access": "not_a_map",
			},
			path: "realm_access.roles",
			want: nil,
		},
		{
			name: "final value is not an array returns nil",
			claims: map[string]interface{}{
				"roles": "admin",
			},
			path: "roles",
			want: nil,
		},
		{
			name: "array with mixed types filters non-strings",
			claims: map[string]interface{}{
				"roles": []interface{}{"admin", 123, "user", nil, "manager"},
			},
			path: "roles",
			want: []string{"admin", "user", "manager"},
		},
		{
			name:   "empty array returns empty slice",
			claims: map[string]interface{}{"roles": []interface{}{}},
			path:   "roles",
			want:   []string{},
		},
		{
			name:   "nil claims returns nil",
			claims: nil,
			path:   "roles",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRoles(tt.claims, tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

type mockOIDCServer struct {
	server     *httptest.Server
	privateKey *rsa.PrivateKey
}

func newMockOIDCServer(t *testing.T) *mockOIDCServer {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	mock := &mockOIDCServer{
		privateKey: privateKey,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		baseURL := mock.server.URL
		config := map[string]interface{}{
			"issuer":                 baseURL,
			"authorization_endpoint": baseURL + "/auth",
			"token_endpoint":         baseURL + "/token",
			"userinfo_endpoint":      baseURL + "/userinfo",
			"jwks_uri":               baseURL + "/jwks",
			"response_types_supported": []string{
				"code",
				"token",
				"id_token",
			},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"kid": "test-key-id",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		idToken := mock.createIDToken(t, map[string]interface{}{
			"sub":         "user-123",
			"email":       "test@example.com",
			"name":        "John Doe",
			"given_name":  "John",
			"family_name": "Doe",
		})

		resp := map[string]interface{}{
			"access_token":  "mock-access-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"id_token":      idToken,
			"refresh_token": "mock-refresh-token",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *mockOIDCServer) createIDToken(t *testing.T, claims map[string]interface{}) string {
	now := time.Now()
	jwtClaims := jwt.MapClaims{
		"iss": m.server.URL,
		"aud": "test-client-id",
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
	}
	for k, v := range claims {
		jwtClaims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims)
	token.Header["kid"] = "test-key-id"

	signed, err := token.SignedString(m.privateKey)
	require.NoError(t, err)
	return signed
}

func (m *mockOIDCServer) Close() {
	m.server.Close()
}

func TestNewProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockOIDCServer(t)
		defer mock.Close()

		cfg := &config.OpenIDConfig{
			ProviderURL:  mock.server.URL,
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:8080/callback",
		}

		provider, err := NewProvider(context.Background(), cfg)

		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("success with custom scopes", func(t *testing.T) {
		mock := newMockOIDCServer(t)
		defer mock.Close()

		cfg := &config.OpenIDConfig{
			ProviderURL:  mock.server.URL,
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:8080/callback",
			Scopes:       []string{"openid", "profile", "email", "roles"},
		}

		provider, err := NewProvider(context.Background(), cfg)

		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("error with invalid provider URL", func(t *testing.T) {
		cfg := &config.OpenIDConfig{
			ProviderURL:  "http://invalid-url-that-does-not-exist.local",
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:8080/callback",
		}

		provider, err := NewProvider(context.Background(), cfg)

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "failed to create OIDC provider")
	})
}

func TestProvider_GetAuthURL(t *testing.T) {
	mock := newMockOIDCServer(t)
	defer mock.Close()

	cfg := &config.OpenIDConfig{
		ProviderURL:  mock.server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)

	authURL := p.GetAuthURL("test-state")

	assert.Contains(t, authURL, mock.server.URL+"/auth")
	assert.Contains(t, authURL, "state=test-state")
	assert.Contains(t, authURL, "client_id=test-client-id")
	assert.Contains(t, authURL, "redirect_uri=")
}

func TestProvider_Exchange(t *testing.T) {
	mock := newMockOIDCServer(t)
	defer mock.Close()

	cfg := &config.OpenIDConfig{
		ProviderURL:  mock.server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)

	token, err := p.Exchange(context.Background(), "valid-code")

	require.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "mock-access-token", token.AccessToken)
	assert.Equal(t, "mock-refresh-token", token.RefreshToken)

	idToken := token.Extra("id_token")
	assert.NotNil(t, idToken)
}

func TestProvider_VerifyIDToken(t *testing.T) {
	mock := newMockOIDCServer(t)
	defer mock.Close()

	cfg := &config.OpenIDConfig{
		ProviderURL:  mock.server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		rawToken := mock.createIDToken(t, map[string]interface{}{
			"sub":   "user-123",
			"email": "test@example.com",
		})

		idToken, err := p.VerifyIDToken(context.Background(), rawToken)

		require.NoError(t, err)
		assert.NotNil(t, idToken)
		assert.Equal(t, "user-123", idToken.Subject)
	})

	t.Run("error with invalid token", func(t *testing.T) {
		idToken, err := p.VerifyIDToken(context.Background(), "invalid-token")

		assert.Error(t, err)
		assert.Nil(t, idToken)
	})
}

func TestProvider_GetUserInfo(t *testing.T) {
	mock := newMockOIDCServer(t)
	defer mock.Close()

	cfg := &config.OpenIDConfig{
		ProviderURL:  mock.server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
		RolesClaim:   "realm_access.roles",
	}

	p, err := NewProvider(context.Background(), cfg)
	require.NoError(t, err)

	t.Run("success with all claims", func(t *testing.T) {
		rawToken := mock.createIDToken(t, map[string]interface{}{
			"sub":         "user-123",
			"email":       "test@example.com",
			"name":        "John Doe",
			"given_name":  "John",
			"family_name": "Doe",
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"admin", "user"},
			},
		})

		idToken, err := p.VerifyIDToken(context.Background(), rawToken)
		require.NoError(t, err)

		userInfo, err := p.GetUserInfo(context.Background(), nil, idToken)

		require.NoError(t, err)
		assert.Equal(t, "user-123", userInfo.Subject)
		assert.Equal(t, "test@example.com", userInfo.Email)
		assert.Equal(t, "John Doe", userInfo.Name)
		assert.Equal(t, "John", userInfo.FirstName)
		assert.Equal(t, "Doe", userInfo.LastName)
		assert.Equal(t, []string{"admin", "user"}, userInfo.Roles)
	})

	t.Run("success with name fallback to firstname/lastname", func(t *testing.T) {
		rawToken := mock.createIDToken(t, map[string]interface{}{
			"sub":  "user-456",
			"name": "Jane Smith",
		})

		idToken, err := p.VerifyIDToken(context.Background(), rawToken)
		require.NoError(t, err)

		userInfo, err := p.GetUserInfo(context.Background(), nil, idToken)

		require.NoError(t, err)
		assert.Equal(t, "user-456", userInfo.Subject)
		assert.Equal(t, "Jane Smith", userInfo.Name)
		assert.Equal(t, "Jane", userInfo.FirstName)
		assert.Equal(t, "Smith", userInfo.LastName)
	})

	t.Run("success with single name", func(t *testing.T) {
		rawToken := mock.createIDToken(t, map[string]interface{}{
			"sub":  "user-789",
			"name": "Madonna",
		})

		idToken, err := p.VerifyIDToken(context.Background(), rawToken)
		require.NoError(t, err)

		userInfo, err := p.GetUserInfo(context.Background(), nil, idToken)

		require.NoError(t, err)
		assert.Equal(t, "Madonna", userInfo.FirstName)
		assert.Equal(t, "", userInfo.LastName)
	})

	t.Run("success without roles claim configured", func(t *testing.T) {
		cfgNoRoles := &config.OpenIDConfig{
			ProviderURL:  mock.server.URL,
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:8080/callback",
		}

		pNoRoles, err := NewProvider(context.Background(), cfgNoRoles)
		require.NoError(t, err)

		rawToken := mock.createIDToken(t, map[string]interface{}{
			"sub":   "user-123",
			"email": "test@example.com",
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"admin"},
			},
		})

		idToken, err := pNoRoles.VerifyIDToken(context.Background(), rawToken)
		require.NoError(t, err)

		userInfo, err := pNoRoles.GetUserInfo(context.Background(), nil, idToken)

		require.NoError(t, err)
		assert.Nil(t, userInfo.Roles)
	})
}
