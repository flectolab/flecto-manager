package jwt

import (
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func testConfig() *config.JWTConfig {
	return &config.JWTConfig{
		Secret:          "test-secret-key-that-is-at-least-32-chars",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test-issuer",
	}
}

func testUser() *model.User {
	return &model.User{
		ID:       123,
		Username: "testuser",
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestNewServiceJWT(t *testing.T) {
	cfg := testConfig()
	service := NewServiceJWT(cfg)

	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
}

func TestServiceJWT_GetSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   []byte
	}{
		{
			name:   "returns secret as bytes",
			secret: "my-secret-key",
			want:   []byte("my-secret-key"),
		},
		{
			name:   "handles empty secret",
			secret: "",
			want:   []byte(""),
		},
		{
			name:   "handles special characters",
			secret: "secret!@#$%^&*()",
			want:   []byte("secret!@#$%^&*()"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Secret = tt.secret
			service := NewServiceJWT(cfg)

			got := service.GetSecret()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServiceJWT_GenerateAccessToken(t *testing.T) {
	cfg := testConfig()
	service := NewServiceJWT(cfg)
	user := testUser()

	token, expiresAt, err := service.GenerateAccessToken(user, types.AuthTypeBasic, nil, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, expiresAt, time.Now().Unix())

	// Verify token content
	claims := &Claims{}
	parsedToken, err := service.parseToken(token, claims)

	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, types.TokenTypeAccess, claims.TokenType)
	assert.Equal(t, cfg.Issuer, claims.Issuer)
}

func TestServiceJWT_GenerateRefreshToken(t *testing.T) {
	cfg := testConfig()
	service := NewServiceJWT(cfg)
	user := testUser()

	token, expiresAt, err := service.GenerateRefreshToken(user, types.AuthTypeBasic, nil, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, expiresAt, time.Now().Unix())

	// Verify token content
	claims := &Claims{}
	parsedToken, err := service.parseToken(token, claims)

	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, types.TokenTypeRefresh, claims.TokenType)
}

func TestServiceJWT_GenerateTokenPair(t *testing.T) {
	cfg := testConfig()
	service := NewServiceJWT(cfg)
	user := testUser()

	tokenPair, err := service.GenerateTokenPair(user, types.AuthTypeBasic, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, tokenPair)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
	assert.Greater(t, tokenPair.ExpiresAt, time.Now().Unix())

	// Verify access token
	accessClaims := &Claims{}
	accessToken, err := service.parseToken(tokenPair.AccessToken, accessClaims)
	assert.NoError(t, err)
	assert.True(t, accessToken.Valid)
	assert.Equal(t, types.TokenTypeAccess, accessClaims.TokenType)

	// Verify refresh token
	refreshClaims := &Claims{}
	refreshToken, err := service.parseToken(tokenPair.RefreshToken, refreshClaims)
	assert.NoError(t, err)
	assert.True(t, refreshToken.Valid)
	assert.Equal(t, types.TokenTypeRefresh, refreshClaims.TokenType)

	// Tokens should be different
	assert.NotEqual(t, tokenPair.AccessToken, tokenPair.RefreshToken)
}

func TestServiceJWT_generateToken(t *testing.T) {
	tests := []struct {
		name      string
		tokenType types.TokenType
		ttl       time.Duration
	}{
		{
			name:      "generates access token",
			tokenType: types.TokenTypeAccess,
			ttl:       15 * time.Minute,
		},
		{
			name:      "generates refresh token",
			tokenType: types.TokenTypeRefresh,
			ttl:       7 * 24 * time.Hour,
		},
		{
			name:      "handles short TTL",
			tokenType: types.TokenTypeAccess,
			ttl:       1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			service := NewServiceJWT(cfg)
			user := testUser()
			permissions := &model.SubjectPermissions{}
			token, expiresAt, err := service.generateToken(user, types.AuthTypeBasic, tt.tokenType, permissions, []string{"role"}, tt.ttl)

			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Check expiration is approximately correct
			expectedExpiry := time.Now().Add(tt.ttl).Unix()
			assert.InDelta(t, expectedExpiry, expiresAt, 2) // Allow 2 second tolerance

			// Verify claims
			claims := &Claims{}
			parsedToken, err := service.parseToken(token, claims)
			assert.NoError(t, err)
			assert.True(t, parsedToken.Valid)
			assert.Equal(t, tt.tokenType, claims.TokenType)
			assert.Equal(t, user.ID, claims.UserID)
			assert.Equal(t, user.Username, claims.Username)
			assert.Equal(t, cfg.Issuer, claims.Issuer)
			assert.Equal(t, user.Username, claims.Subject)
		})
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{
			name:  "hashes token correctly",
			token: "my-test-token",
			want:  "f27c5ab35d1d0c48cbd679025f9f4eec9296b0f8c5cbdadfe6cef6c77787de70",
		},
		{
			name:  "handles empty string",
			token: "",
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashToken(tt.token)

			// Hash should be 64 characters (SHA256 hex)
			assert.Len(t, got, 64)

			// Same input should produce same output
			got2 := HashToken(tt.token)
			assert.Equal(t, got, got2)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	token := "test-token-12345"

	hash1 := HashToken(token)
	hash2 := HashToken(token)
	hash3 := HashToken(token)

	assert.Equal(t, hash1, hash2)
	assert.Equal(t, hash2, hash3)
}

func TestHashToken_UniqueForDifferentInputs(t *testing.T) {
	tokens := []string{
		"token1",
		"token2",
		"token3",
		"Token1", // case sensitive
	}

	hashes := make(map[string]bool)
	for _, token := range tokens {
		hash := HashToken(token)
		assert.False(t, hashes[hash], "hash collision detected for %s", token)
		hashes[hash] = true
	}
}

func TestServiceJWT_parseToken(t *testing.T) {
	cfg := testConfig()
	service := NewServiceJWT(cfg)
	user := testUser()

	t.Run("parses valid token", func(t *testing.T) {
		token, _, err := service.GenerateAccessToken(user, types.AuthTypeBasic, nil, nil)
		assert.NoError(t, err)

		claims := &Claims{}
		parsedToken, err := service.parseToken(token, claims)

		assert.NoError(t, err)
		assert.True(t, parsedToken.Valid)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Username, claims.Username)
	})

	t.Run("fails on invalid token", func(t *testing.T) {
		claims := &Claims{}
		_, err := service.parseToken("invalid-token", claims)

		assert.Error(t, err)
	})

	t.Run("fails on token signed with wrong secret", func(t *testing.T) {
		// Create token with different secret
		otherConfig := testConfig()
		otherConfig.Secret = "different-secret-key-that-is-also-32-chars"
		otherService := NewServiceJWT(otherConfig)

		token, _, err := otherService.GenerateAccessToken(user, types.AuthTypeBasic, nil, nil)
		assert.NoError(t, err)

		// Try to parse with original service
		claims := &Claims{}
		parsedToken, err := service.parseToken(token, claims)

		assert.Error(t, err)
		assert.False(t, parsedToken.Valid)
	})

	t.Run("fails on expired token", func(t *testing.T) {
		// Create token with negative TTL (already expired)
		token, _, err := service.generateToken(user, types.AuthTypeBasic, types.TokenTypeAccess, nil, nil, -1*time.Hour)
		assert.NoError(t, err)

		claims := &Claims{}
		parsedToken, err := service.parseToken(token, claims)

		assert.Error(t, err)
		assert.False(t, parsedToken.Valid)
	})

	t.Run("fails on tampered token", func(t *testing.T) {
		token, _, err := service.GenerateAccessToken(user, types.AuthTypeBasic, nil, nil)
		assert.NoError(t, err)

		// Tamper with the token
		tamperedToken := token[:len(token)-5] + "xxxxx"

		claims := &Claims{}
		parsedToken, err := service.parseToken(tamperedToken, claims)

		assert.Error(t, err)
		if parsedToken != nil {
			assert.False(t, parsedToken.Valid)
		}
	})
}

func TestTokenType_Constants(t *testing.T) {
	assert.Equal(t, types.TokenType("access"), types.TokenTypeAccess)
	assert.Equal(t, types.TokenType("refresh"), types.TokenTypeRefresh)
}

func TestClaims_Structure(t *testing.T) {
	cfg := testConfig()
	service := NewServiceJWT(cfg)
	user := testUser()

	token, _, err := service.GenerateAccessToken(user, types.AuthTypeBasic, nil, nil)
	assert.NoError(t, err)

	claims := &Claims{}
	_, err = service.parseToken(token, claims)
	assert.NoError(t, err)

	// Verify all expected fields are present
	assert.NotZero(t, claims.UserID)
	assert.NotEmpty(t, claims.Username)
	assert.NotEmpty(t, claims.TokenType)
	assert.NotEmpty(t, claims.Issuer)
	assert.NotEmpty(t, claims.Subject)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.ExpiresAt)
}

func TestServiceJWT_TokenExpiration(t *testing.T) {
	cfg := testConfig()
	cfg.AccessTokenTTL = 1 * time.Hour
	cfg.RefreshTokenTTL = 24 * time.Hour
	service := NewServiceJWT(cfg)
	user := testUser()

	t.Run("access token has correct expiration", func(t *testing.T) {
		_, expiresAt, err := service.GenerateAccessToken(user, types.AuthTypeBasic, nil, nil)
		assert.NoError(t, err)

		expectedExpiry := time.Now().Add(cfg.AccessTokenTTL).Unix()
		assert.InDelta(t, expectedExpiry, expiresAt, 2)
	})

	t.Run("refresh token has correct expiration", func(t *testing.T) {
		_, expiresAt, err := service.GenerateRefreshToken(user, types.AuthTypeBasic, nil, nil)
		assert.NoError(t, err)

		expectedExpiry := time.Now().Add(cfg.RefreshTokenTTL).Unix()
		assert.InDelta(t, expectedExpiry, expiresAt, 2)
	})

	t.Run("token pair uses access token expiration", func(t *testing.T) {
		tokenPair, err := service.GenerateTokenPair(user, types.AuthTypeBasic, nil, nil)
		assert.NoError(t, err)

		expectedExpiry := time.Now().Add(cfg.AccessTokenTTL).Unix()
		assert.InDelta(t, expectedExpiry, tokenPair.ExpiresAt, 2)
	})
}

func TestServiceJWT_SigningMethod(t *testing.T) {
	cfg := testConfig()
	service := NewServiceJWT(cfg)
	user := testUser()

	token, _, err := service.GenerateAccessToken(user, types.AuthTypeBasic, nil, nil)
	assert.NoError(t, err)

	// Parse without validation to check signing method
	parsedToken, _ := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return service.GetSecret(), nil
	})

	assert.Equal(t, jwt.SigningMethodHS256.Alg(), parsedToken.Method.Alg())
}
