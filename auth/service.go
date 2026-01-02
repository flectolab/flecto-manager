package auth

import (
	"context"
	"errors"

	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrNoPassword         = errors.New("user has no password set")
)

const (
	BcryptCost = 12
)

type AuthService struct {
	db         *gorm.DB
	jwtService *jwt.ServiceJWT
}

func NewAuthService(db *gorm.DB, jwtService *jwt.ServiceJWT) *AuthService {
	return &AuthService{
		db:         db,
		jwtService: jwtService,
	}
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// Login authenticates a user with password
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*model.User, *jwt.TokenPair, error) {
	user, err := gorm.G[*model.User](s.db).Where("username = ?", req.Username).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if !user.IsActive() {
		return nil, nil, ErrUserInactive
	}

	if !user.HasPassword() {
		return nil, nil, ErrNoPassword
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}
	// Generate tokens
	tokenPair, err := s.jwtService.GenerateTokenPair(user, jwt.AuthTypeBasic, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	// Store refresh token hash
	user.RefreshTokenHash = jwt.HashToken(tokenPair.RefreshToken)
	if err := s.db.WithContext(ctx).Save(user).Error; err != nil {
		return nil, nil, err
	}

	return user, tokenPair, nil
}

// RefreshTokens generates new token pair using a valid refresh token
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string, claims *jwt.Claims) (*model.User, *jwt.TokenPair, error) {
	if claims.TokenType != jwt.TokenTypeRefresh {
		return nil, nil, ErrInvalidCredentials
	}

	user, err := gorm.G[*model.User](s.db).Where("id = ?", claims.UserID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, err
	}

	if !user.IsActive() {
		return nil, nil, ErrUserInactive
	}

	// Verify refresh token hash matches (for revocation check)
	if user.RefreshTokenHash != jwt.HashToken(refreshToken) {
		return nil, nil, ErrInvalidCredentials
	}

	// Generate new tokens
	tokenPair, err := s.jwtService.GenerateTokenPair(user, jwt.AuthTypeBasic, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	// Update refresh token hash
	user.RefreshTokenHash = jwt.HashToken(tokenPair.RefreshToken)
	if err := s.db.WithContext(ctx).Save(user).Error; err != nil {
		return nil, nil, err
	}

	return user, tokenPair, nil
}

// Logout invalidates the user's refresh token
func (s *AuthService) Logout(ctx context.Context, userID int64) error {
	return s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("refresh_token_hash", "").Error
}

// GetUserByID retrieves a user by their ID
func (s *AuthService) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	user, err := gorm.G[*model.User](s.db).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
}
