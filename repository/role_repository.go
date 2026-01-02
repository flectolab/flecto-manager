package repository

import (
	"context"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type RoleRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, role *model.Role) error
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*model.Role, error)
	FindByCode(ctx context.Context, code string) (*model.Role, error)
	FindByCodeAndType(ctx context.Context, code string, roleType model.RoleType) (*model.Role, error)
	FindAll(ctx context.Context) ([]model.Role, error)
	FindAllByType(ctx context.Context, roleType model.RoleType) ([]model.Role, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Role, int64, error)

	// User-Role associations
	AddUserToRole(ctx context.Context, userID, roleID int64) error
	RemoveUserFromRole(ctx context.Context, userID, roleID int64) error
	GetUserRoles(ctx context.Context, userID int64) ([]model.Role, error)
	GetUserRolesByType(ctx context.Context, userID int64, roleType model.RoleType) ([]model.Role, error)
	GetRoleUsers(ctx context.Context, roleID int64) ([]model.User, error)
	GetRoleUsersPaginate(ctx context.Context, roleID int64, search string, limit, offset int) ([]model.User, int64, error)
	GetUsersNotInRole(ctx context.Context, roleID int64, search string, limit int) ([]model.User, error)
	HasUserRole(ctx context.Context, userID, roleID int64) (bool, error)
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *roleRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Role{})
}

func (r *roleRepository) Create(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *roleRepository) Update(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *roleRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete user_roles associations
		if err := tx.Where("role_id = ?", id).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		// Delete role
		return tx.Where("id = ?", id).Delete(&model.Role{}).Error
	})
}

func (r *roleRepository) FindByID(ctx context.Context, id int64) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Preload("Resources").Preload("Admin").Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindByCode(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Preload("Resources").Preload("Admin").Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindByCodeAndType(ctx context.Context, code string, roleType model.RoleType) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Preload("Resources").Preload("Admin").Where("code = ? AND type = ?", code, roleType).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindAll(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Preload("Resources").Preload("Admin").Find(&roles).Error
	return roles, err
}

func (r *roleRepository) FindAllByType(ctx context.Context, roleType model.RoleType) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Preload("Resources").Preload("Admin").Where("type = ?", roleType).Find(&roles).Error
	return roles, err
}

func (r *roleRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Role, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.Role{}).Preload("Resources").Preload("Admin")
	}
	query = query.Preload("Resources").Preload("Admin")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var roles []model.Role
	if err := query.Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

func (r *roleRepository) AddUserToRole(ctx context.Context, userID, roleID int64) error {
	userRole := &model.UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	return r.db.WithContext(ctx).Create(userRole).Error
}

func (r *roleRepository) RemoveUserFromRole(ctx context.Context, userID, roleID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&model.UserRole{}).Error
}

func (r *roleRepository) GetUserRoles(ctx context.Context, userID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Preload("Resources").Preload("Admin").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

func (r *roleRepository) GetUserRolesByType(ctx context.Context, userID int64, roleType model.RoleType) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Preload("Resources").Preload("Admin").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.type = ?", userID, roleType).
		Find(&roles).Error
	return roles, err
}

func (r *roleRepository) GetRoleUsers(ctx context.Context, roleID int64) ([]model.User, error) {
	var users []model.User
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Where("user_roles.role_id = ?", roleID).
		Find(&users).Error
	return users, err
}

func (r *roleRepository) HasUserRole(ctx context.Context, userID, roleID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *roleRepository) GetRoleUsersPaginate(ctx context.Context, roleID int64, search string, limit, offset int) ([]model.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.User{}).
		Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Where("user_roles.role_id = ?", roleID)

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("users.username LIKE ? OR users.firstname LIKE ? OR users.lastname LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var users []model.User
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *roleRepository) GetUsersNotInRole(ctx context.Context, roleID int64, search string, limit int) ([]model.User, error) {
	subQuery := r.db.WithContext(ctx).Model(&model.UserRole{}).
		Select("user_id").
		Where("role_id = ?", roleID)

	query := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id NOT IN (?)", subQuery).
		Where("active = ?", true)

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("username LIKE ? OR firstname LIKE ? OR lastname LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var users []model.User
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}
