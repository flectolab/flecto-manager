package jwt

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID             int64                     `json:"uid"`
	Username           string                    `json:"username"`
	AuthType           types.AuthType            `json:"authType"`
	TokenType          types.TokenType           `json:"type"`
	ExtraRoles         []string                  `json:"roles,omitempty"`
	SubjectPermissions *model.SubjectPermissions `json:"permissions,omitempty"`
}

type ServiceJWT struct {
	config *config.JWTConfig
}

func NewServiceJWT(cfg *config.JWTConfig) *ServiceJWT {
	return &ServiceJWT{config: cfg}
}

// GetSecret returns the JWT secret for use with Echo JWT middleware
func (s *ServiceJWT) GetSecret() []byte {
	return []byte(s.config.Secret)
}

// GenerateTokenPair creates both access and refresh tokens for a user
func (s *ServiceJWT) GenerateTokenPair(user *model.User, authType types.AuthType, subjectPermissions *model.SubjectPermissions, extraRoles []string) (*types.TokenPair, error) {
	accessToken, expiresAt, err := s.generateToken(user, authType, types.TokenTypeAccess, subjectPermissions, extraRoles, s.config.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, _, err := s.generateToken(user, authType, types.TokenTypeRefresh, subjectPermissions, extraRoles, s.config.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &types.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// GenerateAccessToken creates only an access token for a user
func (s *ServiceJWT) GenerateAccessToken(user *model.User, authType types.AuthType, subjectPermissions *model.SubjectPermissions, extraRoles []string) (string, int64, error) {
	return s.generateToken(user, authType, types.TokenTypeAccess, subjectPermissions, extraRoles, s.config.AccessTokenTTL)
}

// GenerateRefreshToken creates only a refresh token for a user
func (s *ServiceJWT) GenerateRefreshToken(user *model.User, authType types.AuthType, subjectPermissions *model.SubjectPermissions, extraRoles []string) (string, int64, error) {
	return s.generateToken(user, authType, types.TokenTypeRefresh, subjectPermissions, extraRoles, s.config.RefreshTokenTTL)
}

func (s *ServiceJWT) generateToken(user *model.User, authType types.AuthType, tokenType types.TokenType, subjectPermissions *model.SubjectPermissions, extraRoles []string, ttl time.Duration) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		UserID:             user.ID,
		Username:           user.Username,
		TokenType:          tokenType,
		AuthType:           authType,
		SubjectPermissions: subjectPermissions,
	}
	if len(extraRoles) > 0 {
		claims.ExtraRoles = extraRoles
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.GetSecret())
	if err != nil {
		return "", 0, err
	}

	return signedToken, expiresAt.Unix(), nil
}

// HashToken creates a SHA256 hash of a token for secure storage
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// parseToken parses a JWT token and extracts the claims
func (s *ServiceJWT) parseToken(tokenString string, claims *Claims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.GetSecret(), nil
	})
}
