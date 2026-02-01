package service

import (
	"context"
	"errors"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/hash"
	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/types"
	"gorm.io/gorm"
)

type AuthService interface {
	Login(ctx context.Context, req *types.LoginRequest) (*model.User, *types.TokenPair, error)
	RefreshTokens(ctx context.Context, refreshToken string, claims *jwt.Claims) (*model.User, *types.TokenPair, error)
	Logout(ctx context.Context, userID int64) error
	ToUserResponse(user *model.User) *types.UserResponse
}

type authService struct {
	ctx        *appContext.Context
	userRepo   repository.UserRepository
	jwtService *jwt.ServiceJWT
}

func NewAuthService(ctx *appContext.Context, userRepo repository.UserRepository, jwtService *jwt.ServiceJWT) AuthService {
	return &authService{
		ctx:        ctx,
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

// Login authenticates a user with password
func (s *authService) Login(ctx context.Context, req *types.LoginRequest) (*model.User, *types.TokenPair, error) {
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.ctx.Logger.Warn("login failed: user not found", "username", req.Username)
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if !user.IsActive() || !user.HasPassword() {
		s.ctx.Logger.Warn("login failed: user inactive or has no password", "username", req.Username)
		return nil, nil, ErrUserNotFound
	}

	if err = hash.CheckPassword(user.Password, req.Password); err != nil {
		s.ctx.Logger.Warn("login failed: invalid password", "username", req.Username)
		return nil, nil, ErrInvalidCredentials
	}
	// Generate tokens
	tokenPair, err := s.jwtService.GenerateTokenPair(user, types.AuthTypeBasic, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	// Store refresh token hash
	user.RefreshTokenHash = jwt.HashToken(tokenPair.RefreshToken)

	if err = s.userRepo.Update(ctx, user); err != nil {
		return nil, nil, err
	}

	s.ctx.Logger.Info("user logged in", "username", req.Username, "id", user.ID)
	return user, tokenPair, nil
}

// RefreshTokens generates new token pair using a valid refresh token
func (s *authService) RefreshTokens(ctx context.Context, refreshToken string, claims *jwt.Claims) (*model.User, *types.TokenPair, error) {
	if claims.TokenType != types.TokenTypeRefresh {
		return nil, nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
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
	tokenPair, err := s.jwtService.GenerateTokenPair(user, types.AuthTypeBasic, claims.SubjectPermissions, claims.ExtraRoles)
	if err != nil {
		return nil, nil, err
	}

	// Update refresh token hash
	user.RefreshTokenHash = jwt.HashToken(tokenPair.RefreshToken)
	if err = s.userRepo.GetQuery(ctx).Where("id = ?", user.ID).UpdateColumn("refresh_token_hash", user.RefreshTokenHash).Error; err != nil {
		return nil, nil, err
	}

	return user, tokenPair, nil
}

// Logout invalidates the user's refresh token
func (s *authService) Logout(ctx context.Context, userID int64) error {
	return s.userRepo.GetQuery(ctx).Where("id = ?", userID).Update("refresh_token_hash", "").Error
}

func (s *authService) ToUserResponse(user *model.User) *types.UserResponse {
	return &types.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
	}
}
