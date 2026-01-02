package service

import (
	"context"
	"errors"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"gorm.io/gorm"
)

var (
	ErrRoleNotFound      = errors.New("role not found")
	ErrRoleAlreadyExists = errors.New("role already exists")
	ErrUserNotInRole     = errors.New("user is not in role")
	ErrUserAlreadyInRole = errors.New("user is already in role")
)

type RoleService interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, input *model.Role) (*model.Role, error)
	Update(ctx context.Context, id int64, input model.Role) (*model.Role, error)
	Delete(ctx context.Context, id int64) (bool, error)
	GetByID(ctx context.Context, id int64) (*model.Role, error)
	GetByCode(ctx context.Context, code string, roleType model.RoleType) (*model.Role, error)
	GetAll(ctx context.Context) ([]model.Role, error)
	GetAllByType(ctx context.Context, roleType model.RoleType) ([]model.Role, error)
	SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.RoleList, error)

	// User-Role management
	AddUserToRole(ctx context.Context, userID, roleID int64) error
	RemoveUserFromRole(ctx context.Context, userID, roleID int64) error
	GetUserRoles(ctx context.Context, userID int64) ([]model.Role, error)
	GetUserRolesByType(ctx context.Context, userID int64, roleType model.RoleType) ([]model.Role, error)
	GetRoleUsers(ctx context.Context, roleID int64) ([]model.User, error)
	GetRoleUsersPaginate(ctx context.Context, roleCode string, pagination *commonTypes.PaginationInput, search string) (*model.UserList, error)
	GetUsersNotInRole(ctx context.Context, roleCode string, search string, limit int) ([]model.User, error)

	// Permissions
	GetPermissionsByRoleCode(ctx context.Context, code string) (*model.SubjectPermissions, error)
	GetPermissionsByUsername(ctx context.Context, username string) (*model.SubjectPermissions, error)
	GetPermissionsByTokenName(ctx context.Context, tokenName string) (*model.SubjectPermissions, error)
	UpdateRolePermissions(ctx context.Context, roleID int64, permissions *model.SubjectPermissions) error
	UpdateUserRoles(ctx context.Context, userID int64, roleCodes []string) error
}

type roleService struct {
	ctx      *appContext.Context
	repo     repository.RoleRepository
	userRepo repository.UserRepository
}

func NewRoleService(
	ctx *appContext.Context,
	repo repository.RoleRepository,
	userRepo repository.UserRepository,
) RoleService {
	return &roleService{
		ctx:      ctx,
		repo:     repo,
		userRepo: userRepo,
	}
}

func (s *roleService) GetTx(ctx context.Context) *gorm.DB {
	return s.repo.GetTx(ctx)
}

func (s *roleService) GetQuery(ctx context.Context) *gorm.DB {
	return s.repo.GetQuery(ctx)
}

func (s *roleService) Create(ctx context.Context, input *model.Role) (*model.Role, error) {
	// Check if code already exists for this type
	existing, err := s.repo.FindByCodeAndType(ctx, input.Code, input.Type)
	if err == nil && existing != nil {
		return nil, ErrRoleAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	err = s.ctx.Validator.Struct(input)
	if err != nil {
		return nil, err
	}

	if err = s.repo.Create(ctx, input); err != nil {
		return nil, err
	}

	return input, nil
}

func (s *roleService) Update(ctx context.Context, id int64, input model.Role) (*model.Role, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	role.Code = input.Code
	role.Type = input.Type
	err = s.ctx.Validator.Struct(role)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Update(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (s *roleService) Delete(ctx context.Context, id int64) (bool, error) {
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrRoleNotFound
		}
		return false, err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

func (s *roleService) GetByID(ctx context.Context, id int64) (*model.Role, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

func (s *roleService) GetByCode(ctx context.Context, code string, roleType model.RoleType) (*model.Role, error) {
	role, err := s.repo.FindByCodeAndType(ctx, code, roleType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

func (s *roleService) GetAll(ctx context.Context) ([]model.Role, error) {
	return s.repo.FindAll(ctx)
}

func (s *roleService) GetAllByType(ctx context.Context, roleType model.RoleType) ([]model.Role, error) {
	return s.repo.FindAllByType(ctx, roleType)
}

func (s *roleService) SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.RoleList, error) {
	roles, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.RoleList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  roles,
	}, nil
}

func (s *roleService) AddUserToRole(ctx context.Context, userID, roleID int64) error {
	// Check role exists
	_, err := s.repo.FindByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}

	// Check if user already has the role
	hasRole, err := s.repo.HasUserRole(ctx, userID, roleID)
	if err != nil {
		return err
	}
	if hasRole {
		return ErrUserAlreadyInRole
	}

	return s.repo.AddUserToRole(ctx, userID, roleID)
}

func (s *roleService) RemoveUserFromRole(ctx context.Context, userID, roleID int64) error {
	// Check if user has the role
	hasRole, err := s.repo.HasUserRole(ctx, userID, roleID)
	if err != nil {
		return err
	}
	if !hasRole {
		return ErrUserNotInRole
	}

	return s.repo.RemoveUserFromRole(ctx, userID, roleID)
}

func (s *roleService) GetUserRoles(ctx context.Context, userID int64) ([]model.Role, error) {
	return s.repo.GetUserRoles(ctx, userID)
}

func (s *roleService) GetUserRolesByType(ctx context.Context, userID int64, roleType model.RoleType) ([]model.Role, error) {
	return s.repo.GetUserRolesByType(ctx, userID, roleType)
}

func (s *roleService) GetRoleUsers(ctx context.Context, roleID int64) ([]model.User, error) {
	_, err := s.repo.FindByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	return s.repo.GetRoleUsers(ctx, roleID)
}

func (s *roleService) GetRoleUsersPaginate(ctx context.Context, roleCode string, pagination *commonTypes.PaginationInput, search string) (*model.UserList, error) {
	role, err := s.repo.FindByCodeAndType(ctx, roleCode, model.RoleTypeRole)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	users, total, err := s.repo.GetRoleUsersPaginate(ctx, role.ID, search, pagination.GetLimit(), pagination.GetOffset())
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

func (s *roleService) GetUsersNotInRole(ctx context.Context, roleCode string, search string, limit int) ([]model.User, error) {
	role, err := s.repo.FindByCodeAndType(ctx, roleCode, model.RoleTypeRole)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	return s.repo.GetUsersNotInRole(ctx, role.ID, search, limit)
}

func (s *roleService) GetPermissionsByRoleCode(ctx context.Context, code string) (*model.SubjectPermissions, error) {
	role, err := s.repo.FindByCodeAndType(ctx, code, model.RoleTypeRole)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	return &model.SubjectPermissions{
		Resources: role.Resources,
		Admin:     role.Admin,
	}, nil
}

func (s *roleService) GetPermissionsByUsername(ctx context.Context, username string) (*model.SubjectPermissions, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	if len(roles) == 0 {
		return &model.SubjectPermissions{
			Resources: []model.ResourcePermission{},
			Admin:     []model.AdminPermission{},
		}, nil
	}
	resources := make([]model.ResourcePermission, 0)
	admin := make([]model.AdminPermission, 0)
	for _, role := range roles {
		resources = append(resources, role.Resources...)
		admin = append(admin, role.Admin...)
	}

	return &model.SubjectPermissions{
		Resources: deduplicateResourcePermissions(resources),
		Admin:     deduplicateAdminPermissions(admin),
	}, nil
}

func (s *roleService) GetPermissionsByTokenName(ctx context.Context, tokenName string) (*model.SubjectPermissions, error) {
	roleCode := "token_" + tokenName
	role, err := s.repo.FindByCodeAndType(ctx, roleCode, model.RoleTypeToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.SubjectPermissions{
				Resources: []model.ResourcePermission{},
				Admin:     []model.AdminPermission{},
			}, nil
		}
		return nil, err
	}

	return &model.SubjectPermissions{
		Resources: role.Resources,
		Admin:     role.Admin,
	}, nil
}

func deduplicateResourcePermissions(perms []model.ResourcePermission) []model.ResourcePermission {
	seen := make(map[string]struct{})
	result := make([]model.ResourcePermission, 0, len(perms))

	for _, p := range perms {
		key := p.Namespace + "|" + p.Project + "|" + string(p.Resource) + "|" + string(p.Action)
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, p)
		}
	}

	return result
}

func deduplicateAdminPermissions(perms []model.AdminPermission) []model.AdminPermission {
	seen := make(map[string]struct{})
	result := make([]model.AdminPermission, 0, len(perms))

	for _, p := range perms {
		key := string(p.Section) + "|" + string(p.Action)
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, p)
		}
	}

	return result
}

func (s *roleService) UpdateRolePermissions(ctx context.Context, roleID int64, permissions *model.SubjectPermissions) error {
	_, err := s.repo.FindByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}

	return s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete all existing resource permissions for this role
		if err = tx.Where("role_id = ?", roleID).Delete(&model.ResourcePermission{}).Error; err != nil {
			return err
		}

		// Delete all existing admin permissions for this role
		if err = tx.Where("role_id = ?", roleID).Delete(&model.AdminPermission{}).Error; err != nil {
			return err
		}

		// Create new resource permissions
		if len(permissions.Resources) > 0 {
			resourcePerms := make([]model.ResourcePermission, len(permissions.Resources))
			for i, r := range permissions.Resources {
				resourcePerms[i] = model.ResourcePermission{
					RoleID:    roleID,
					Namespace: r.Namespace,
					Project:   r.Project,
					Resource:  r.Resource,
					Action:    r.Action,
				}
			}
			if err = tx.Create(&resourcePerms).Error; err != nil {
				return err
			}
		}

		// Create new admin permissions
		if len(permissions.Admin) > 0 {
			adminPerms := make([]model.AdminPermission, len(permissions.Admin))
			for i, a := range permissions.Admin {
				adminPerms[i] = model.AdminPermission{
					RoleID:  roleID,
					Section: a.Section,
					Action:  a.Action,
				}
			}
			if err = tx.Create(&adminPerms).Error; err != nil {
				return err
			}
		}

		// Update role's updatedAt timestamp
		if err = tx.Model(&model.Role{}).Where("id = ?", roleID).Update("updated_at", time.Now()).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *roleService) UpdateUserRoles(ctx context.Context, userID int64, roleCodes []string) error {
	// Verify user exists
	_, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// Resolve role codes to IDs (only named roles, not user personal roles)
	roleIDs := make([]int64, 0, len(roleCodes))
	for _, code := range roleCodes {
		role, err := s.repo.FindByCodeAndType(ctx, code, model.RoleTypeRole)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRoleNotFound
			}
			return err
		}
		roleIDs = append(roleIDs, role.ID)
	}

	return s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete all existing user-role associations for this user

		if err = tx.Where("user_id = ? AND role_id IN (?)",
			userID,
			tx.Model(&model.UserRole{}).
				Joins("JOIN roles ON user_roles.role_id = roles.id").
				Select("user_roles.role_id").
				Where("user_roles.user_id = ? AND roles.type = ?", userID, model.RoleTypeRole),
		).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}

		// Create new user-role associations
		if len(roleIDs) > 0 {
			userRoles := make([]model.UserRole, len(roleIDs))
			for i, roleID := range roleIDs {
				userRoles[i] = model.UserRole{
					UserID: userID,
					RoleID: roleID,
				}
			}
			if err = tx.Create(&userRoles).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
