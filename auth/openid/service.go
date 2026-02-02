package openid

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	flectoService "github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
)

var (
	ErrInvalidState = errors.New("invalid state parameter")
	ErrUserInactive = errors.New("user account is inactive")
)

type Service interface {
	BeginAuth() (authURL string, state string, err error)
	CompleteAuth(ctx context.Context, code, state, expectedState string) (*model.User, *types.TokenPair, error)
}

type service struct {
	provider    Provider
	userService flectoService.UserService
	jwtService  *jwt.ServiceJWT
}

func NewService(provider Provider, userService flectoService.UserService, jwtService *jwt.ServiceJWT) Service {
	return &service{
		provider:    provider,
		userService: userService,
		jwtService:  jwtService,
	}
}

func (s *service) BeginAuth() (string, string, error) {
	state, err := generateState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	authURL := s.provider.GetAuthURL(state)
	return authURL, state, nil
}

func (s *service) CompleteAuth(ctx context.Context, code, state, expectedState string) (*model.User, *types.TokenPair, error) {
	if state != expectedState {
		return nil, nil, ErrInvalidState
	}

	token, err := s.provider.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, nil, fmt.Errorf("no id_token in response")
	}

	idToken, err := s.provider.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	userInfo, err := s.provider.GetUserInfo(ctx, token, idToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	user, err := s.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, nil, err
	}

	if !user.IsActive() {
		return nil, nil, ErrUserInactive
	}

	tokenPair, err := s.jwtService.GenerateTokenPair(user, types.AuthTypeOpenID, nil, userInfo.Roles)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	refreshTokenHash := jwt.HashToken(tokenPair.RefreshToken)
	if err = s.userService.UpdateRefreshToken(ctx, user.ID, refreshTokenHash); err != nil {
		return nil, nil, fmt.Errorf("failed to update user: %w", err)
	}
	user.RefreshTokenHash = refreshTokenHash

	return user, tokenPair, nil
}

func (s *service) findOrCreateUser(ctx context.Context, info *UserInfo) (*model.User, error) {
	username := info.Email
	if username == "" {
		username = info.Subject
	}

	active := true
	input := &model.User{
		Username:  username,
		Firstname: info.FirstName,
		Lastname:  info.LastName,
		Active:    &active,
	}

	user, err := s.userService.FindOrCreate(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	return user, nil
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func ToUserResponse(user *model.User) *types.UserResponse {
	return &types.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
	}
}
