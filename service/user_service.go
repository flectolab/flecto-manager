package service

import (
	"context"
	"errors"

	"github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const BcryptCost = 12

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserService interface {
	Create(ctx context.Context, input *model.User) (*model.User, error)
	Update(ctx context.Context, id int64, input model.User) (*model.User, error)
	Delete(ctx context.Context, id int64) (bool, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetAll(ctx context.Context) ([]model.User, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.User, error)
	SearchPaginate(ctx context.Context, pagination *types.PaginationInput, query *gorm.DB) (*model.UserList, error)
	UpdatePassword(ctx context.Context, id int64, newPassword string) error
	UpdateStatus(ctx context.Context, id int64, active bool) (*model.User, error)
	SetPassword(ctx context.Context, id int64, newPassword string) error
}

type userService struct {
	validator *validator.Validate
	repo      repository.UserRepository
	roleRepo  repository.RoleRepository
}

func NewUserService(
	validator *validator.Validate,
	repo repository.UserRepository,
	roleRepo repository.RoleRepository,
) UserService {
	return &userService{
		validator: validator,
		repo:      repo,
		roleRepo:  roleRepo,
	}
}

func (s *userService) Create(ctx context.Context, input *model.User) (*model.User, error) {
	// Check if username already exists
	existing, err := s.repo.FindByUsername(ctx, input.Username)
	if err == nil && existing != nil {
		return nil, ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	err = s.validator.Struct(input)
	if err != nil {
		return nil, err
	}

	if err = s.repo.Create(ctx, input); err != nil {
		return nil, err
	}

	role := &model.Role{Code: input.Username, Type: model.RoleTypeUser}
	err = s.roleRepo.Create(ctx, role)
	if err != nil {
		return nil, err
	}

	err = s.roleRepo.AddUserToRole(ctx, input.ID, role.ID)
	if err != nil {
		return nil, err
	}

	return input, nil
}

func (s *userService) Update(ctx context.Context, id int64, input model.User) (*model.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	user.Firstname = input.Firstname
	user.Lastname = input.Lastname
	err = s.validator.Struct(user)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) Delete(ctx context.Context, id int64) (bool, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return false, err
	}
	role, errGetRole := s.roleRepo.FindByCodeAndType(ctx, user.Username, model.RoleTypeUser)
	if errGetRole != nil {
		return false, errGetRole
	}

	if err = s.roleRepo.Delete(ctx, role.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	if err = s.repo.Delete(ctx, id); err != nil {
		return false, err
	}

	return true, nil
}

func (s *userService) GetByID(ctx context.Context, id int64) (*model.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *userService) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *userService) GetAll(ctx context.Context) ([]model.User, error) {
	return s.repo.FindAll(ctx)
}

func (s *userService) Search(ctx context.Context, query *gorm.DB) ([]model.User, error) {
	return s.repo.Search(ctx, query)
}

func (s *userService) SearchPaginate(ctx context.Context, pagination *types.PaginationInput, query *gorm.DB) (*model.UserList, error) {
	users, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.UserList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  users,
	}, nil
}

func (s *userService) UpdatePassword(ctx context.Context, id int64, newPassword string) error {
	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), BcryptCost)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(ctx, id, string(hashedPassword))
}

func (s *userService) UpdateStatus(ctx context.Context, id int64, active bool) (*model.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if err = s.repo.UpdateStatus(ctx, id, active); err != nil {
		return nil, err
	}

	user.Active = &active
	return user, nil
}

func (s *userService) SetPassword(ctx context.Context, id int64, newPassword string) error {
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), BcryptCost)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(ctx, id, string(hashedPassword))
}
