package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"gorm.io/gorm"
)

var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrTokenAlreadyExists = errors.New("token with this name already exists")
	ErrTokenExpired       = errors.New("token has expired")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenNameTooLong   = errors.New("token name is too long")
)

type TokenService interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, name string, expiresAt *string, permissions *model.SubjectPermissions) (*model.Token, string, error)
	Delete(ctx context.Context, id int64) (bool, error)
	GetByID(ctx context.Context, id int64) (*model.Token, error)
	GetByName(ctx context.Context, name string) (*model.Token, error)
	ValidateToken(ctx context.Context, plainToken string) (*model.Token, *model.SubjectPermissions, error)
	GetAll(ctx context.Context) ([]model.Token, error)
	SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.TokenList, error)
	GetRole(ctx context.Context, tokenID int64) (*model.Role, error)
}

type tokenService struct {
	ctx      *appContext.Context
	repo     repository.TokenRepository
	roleRepo repository.RoleRepository
}

func NewTokenService(
	ctx *appContext.Context,
	repo repository.TokenRepository,
	roleRepo repository.RoleRepository,
) TokenService {
	return &tokenService{
		ctx:      ctx,
		repo:     repo,
		roleRepo: roleRepo,
	}
}

func (s *tokenService) GetTx(ctx context.Context) *gorm.DB {
	return s.repo.GetTx(ctx)
}

func (s *tokenService) GetQuery(ctx context.Context) *gorm.DB {
	return s.repo.GetQuery(ctx)
}

func (s *tokenService) Create(ctx context.Context, name string, expiresAt *string, permissions *model.SubjectPermissions) (*model.Token, string, error) {
	// Validate name length
	if len(name) > model.TokenNameMaxLength {
		return nil, "", ErrTokenNameTooLong
	}

	// Check if token with this name already exists
	existing, err := s.repo.FindByName(ctx, name)
	if err == nil && existing != nil {
		return nil, "", ErrTokenAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}

	// Generate random token
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, "", err
	}
	plainToken := model.TokenPrefix + base64.RawURLEncoding.EncodeToString(randomBytes)

	// Hash the token and create preview
	tokenHash := jwt.HashToken(plainToken)
	tokenPreview := model.GenerateTokenPreview(plainToken)

	// Parse expiration date if provided
	token := &model.Token{
		Name:         name,
		TokenHash:    tokenHash,
		TokenPreview: tokenPreview,
	}

	if expiresAt != nil && *expiresAt != "" {
		parsedTime, err := parseDateTime(*expiresAt)
		if err != nil {
			return nil, "", err
		}
		token.ExpiresAt = &parsedTime
	}

	// Validate the token
	if err = s.ctx.Validator.Struct(token); err != nil {
		return nil, "", err
	}

	// Create token in transaction along with its personal role and permissions
	err = s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		// Create the token
		if err := tx.Create(token).Error; err != nil {
			return err
		}

		// Create the personal role for this token
		role := &model.Role{
			Code: token.GetRoleCode(),
			Type: model.RoleTypeToken,
		}
		if err := tx.Create(role).Error; err != nil {
			return err
		}

		// Add permissions if provided
		if permissions != nil {
			for _, perm := range permissions.Resources {
				resourcePerm := model.ResourcePermission{
					RoleID:    role.ID,
					Namespace: perm.Namespace,
					Project:   perm.Project,
					Resource:  perm.Resource,
					Action:    perm.Action,
				}
				if err := tx.Create(&resourcePerm).Error; err != nil {
					return err
				}
			}
			for _, perm := range permissions.Admin {
				adminPerm := model.AdminPermission{
					RoleID:  role.ID,
					Section: perm.Section,
					Action:  perm.Action,
				}
				if err := tx.Create(&adminPerm).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, "", err
	}

	return token, plainToken, nil
}

func (s *tokenService) Delete(ctx context.Context, id int64) (bool, error) {
	token, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrTokenNotFound
		}
		return false, err
	}

	// Delete token and its personal role in transaction
	err = s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		// Find and delete the personal role
		roleCode := token.GetRoleCode()
		var role model.Role
		if err := tx.Where("code = ? AND type = ?", roleCode, model.RoleTypeToken).First(&role).Error; err == nil {
			// Delete permissions associated with this role
			if err := tx.Where("role_id = ?", role.ID).Delete(&model.ResourcePermission{}).Error; err != nil {
				return err
			}
			if err := tx.Where("role_id = ?", role.ID).Delete(&model.AdminPermission{}).Error; err != nil {
				return err
			}
			// Delete the role
			if err := tx.Delete(&role).Error; err != nil {
				return err
			}
		}

		// Delete the token
		if err := tx.Delete(token).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *tokenService) GetByID(ctx context.Context, id int64) (*model.Token, error) {
	token, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return token, nil
}

func (s *tokenService) GetByName(ctx context.Context, name string) (*model.Token, error) {
	token, err := s.repo.FindByName(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return token, nil
}

func (s *tokenService) ValidateToken(ctx context.Context, plainToken string) (*model.Token, *model.SubjectPermissions, error) {
	// Check prefix
	if len(plainToken) < len(model.TokenPrefix) || plainToken[:len(model.TokenPrefix)] != model.TokenPrefix {
		return nil, nil, ErrInvalidToken
	}

	// Hash the token and look it up
	tokenHash := jwt.HashToken(plainToken)
	token, err := s.repo.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInvalidToken
		}
		return nil, nil, err
	}

	// Check expiration
	if token.IsExpired() {
		return nil, nil, ErrTokenExpired
	}

	// Get the personal role and its permissions
	role, err := s.roleRepo.FindByCodeAndType(ctx, token.GetRoleCode(), model.RoleTypeToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No role found, return empty permissions
			return token, &model.SubjectPermissions{
				Resources: []model.ResourcePermission{},
				Admin:     []model.AdminPermission{},
			}, nil
		}
		return nil, nil, err
	}

	return token, &model.SubjectPermissions{
		Resources: role.Resources,
		Admin:     role.Admin,
	}, nil
}

func (s *tokenService) GetAll(ctx context.Context) ([]model.Token, error) {
	return s.repo.FindAll(ctx)
}

func (s *tokenService) SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.TokenList, error) {
	tokens, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.TokenList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  tokens,
	}, nil
}

func (s *tokenService) GetRole(ctx context.Context, tokenID int64) (*model.Role, error) {
	token, err := s.repo.FindByID(ctx, tokenID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	role, err := s.roleRepo.FindByCodeAndType(ctx, token.GetRoleCode(), model.RoleTypeToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	return role, nil
}

// parseDateTime parses a datetime string in RFC3339 format
func parseDateTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
