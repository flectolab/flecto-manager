package repository

import (
	"context"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type ResourcePermissionRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, perm *model.ResourcePermission) error
	Update(ctx context.Context, perm *model.ResourcePermission) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*model.ResourcePermission, error)
	FindByRoleID(ctx context.Context, roleID int64) ([]model.ResourcePermission, error)
	FindByRoleIDs(ctx context.Context, roleIDs []int64) ([]model.ResourcePermission, error)
	FindByNamespace(ctx context.Context, namespace string) ([]model.ResourcePermission, error)
	FindByNamespaceAndProject(ctx context.Context, namespace, project string) ([]model.ResourcePermission, error)
	DeleteByRoleID(ctx context.Context, roleID int64) error
}

type resourcePermissionRepository struct {
	db *gorm.DB
}

func NewResourcePermissionRepository(db *gorm.DB) ResourcePermissionRepository {
	return &resourcePermissionRepository{db: db}
}

func (r *resourcePermissionRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *resourcePermissionRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.ResourcePermission{})
}

func (r *resourcePermissionRepository) Create(ctx context.Context, perm *model.ResourcePermission) error {
	return r.db.WithContext(ctx).Create(perm).Error
}

func (r *resourcePermissionRepository) Update(ctx context.Context, perm *model.ResourcePermission) error {
	return r.db.WithContext(ctx).Save(perm).Error
}

func (r *resourcePermissionRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ResourcePermission{}).Error
}

func (r *resourcePermissionRepository) FindByID(ctx context.Context, id int64) (*model.ResourcePermission, error) {
	var perm model.ResourcePermission
	err := r.db.WithContext(ctx).Preload("Role").Where("id = ?", id).First(&perm).Error
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

func (r *resourcePermissionRepository) FindByRoleID(ctx context.Context, roleID int64) ([]model.ResourcePermission, error) {
	var perms []model.ResourcePermission
	err := r.db.WithContext(ctx).Preload("Role").Where("role_id = ?", roleID).Find(&perms).Error
	return perms, err
}

func (r *resourcePermissionRepository) FindByRoleIDs(ctx context.Context, roleIDs []int64) ([]model.ResourcePermission, error) {
	var perms []model.ResourcePermission
	if len(roleIDs) == 0 {
		return perms, nil
	}
	err := r.db.WithContext(ctx).Preload("Role").Where("role_id IN ?", roleIDs).Find(&perms).Error
	return perms, err
}

func (r *resourcePermissionRepository) FindByNamespace(ctx context.Context, namespace string) ([]model.ResourcePermission, error) {
	var perms []model.ResourcePermission
	err := r.db.WithContext(ctx).Preload("Role").Where("namespace = ?", namespace).Find(&perms).Error
	return perms, err
}

func (r *resourcePermissionRepository) FindByNamespaceAndProject(ctx context.Context, namespace, project string) ([]model.ResourcePermission, error) {
	var perms []model.ResourcePermission
	err := r.db.WithContext(ctx).Preload("Role").
		Where("(namespace = ? OR namespace = '*') AND (project = ? OR project = '*' OR project = '')", namespace, project).
		Find(&perms).Error
	return perms, err
}

func (r *resourcePermissionRepository) DeleteByRoleID(ctx context.Context, roleID int64) error {
	return r.db.WithContext(ctx).Where("role_id = ?", roleID).Delete(&model.ResourcePermission{}).Error
}

// AdminPermissionRepository

type AdminPermissionRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, perm *model.AdminPermission) error
	Update(ctx context.Context, perm *model.AdminPermission) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*model.AdminPermission, error)
	FindByRoleID(ctx context.Context, roleID int64) ([]model.AdminPermission, error)
	FindByRoleIDs(ctx context.Context, roleIDs []int64) ([]model.AdminPermission, error)
	FindBySection(ctx context.Context, section model.SectionType) ([]model.AdminPermission, error)
	DeleteByRoleID(ctx context.Context, roleID int64) error
}

type adminPermissionRepository struct {
	db *gorm.DB
}

func NewAdminPermissionRepository(db *gorm.DB) AdminPermissionRepository {
	return &adminPermissionRepository{db: db}
}

func (r *adminPermissionRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *adminPermissionRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.AdminPermission{})
}

func (r *adminPermissionRepository) Create(ctx context.Context, perm *model.AdminPermission) error {
	return r.db.WithContext(ctx).Create(perm).Error
}

func (r *adminPermissionRepository) Update(ctx context.Context, perm *model.AdminPermission) error {
	return r.db.WithContext(ctx).Save(perm).Error
}

func (r *adminPermissionRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.AdminPermission{}).Error
}

func (r *adminPermissionRepository) FindByID(ctx context.Context, id int64) (*model.AdminPermission, error) {
	var perm model.AdminPermission
	err := r.db.WithContext(ctx).Preload("Role").Where("id = ?", id).First(&perm).Error
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

func (r *adminPermissionRepository) FindByRoleID(ctx context.Context, roleID int64) ([]model.AdminPermission, error) {
	var perms []model.AdminPermission
	err := r.db.WithContext(ctx).Preload("Role").Where("role_id = ?", roleID).Find(&perms).Error
	return perms, err
}

func (r *adminPermissionRepository) FindByRoleIDs(ctx context.Context, roleIDs []int64) ([]model.AdminPermission, error) {
	var perms []model.AdminPermission
	if len(roleIDs) == 0 {
		return perms, nil
	}
	err := r.db.WithContext(ctx).Preload("Role").Where("role_id IN ?", roleIDs).Find(&perms).Error
	return perms, err
}

func (r *adminPermissionRepository) FindBySection(ctx context.Context, section model.SectionType) ([]model.AdminPermission, error) {
	var perms []model.AdminPermission
	err := r.db.WithContext(ctx).Preload("Role").Where("section = ?", section).Find(&perms).Error
	return perms, err
}

func (r *adminPermissionRepository) DeleteByRoleID(ctx context.Context, roleID int64) error {
	return r.db.WithContext(ctx).Where("role_id = ?", roleID).Delete(&model.AdminPermission{}).Error
}
