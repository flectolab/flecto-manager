package repository

import (
	"context"
	"fmt"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type ProjectCdnConfigRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.ProjectCdnConfig, error)
	// FindEnabledByProject returns only the configurations a publish has to purge
	// through. It is the query on the publish path, so it is kept separate from the
	// listing one rather than filtered by the caller.
	FindEnabledByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.ProjectCdnConfig, error)
	FindByName(ctx context.Context, namespaceCode, projectCode, name string) (*model.ProjectCdnConfig, error)
	Create(ctx context.Context, config *model.ProjectCdnConfig) error
	Update(ctx context.Context, config *model.ProjectCdnConfig) error
	Delete(ctx context.Context, namespaceCode, projectCode, name string) error
}

type projectCdnConfigRepository struct {
	db *gorm.DB
}

func NewProjectCdnConfigRepository(db *gorm.DB) ProjectCdnConfigRepository {
	return &projectCdnConfigRepository{db: db}
}

func (r *projectCdnConfigRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *projectCdnConfigRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.ProjectCdnConfig{})
}

func (r *projectCdnConfigRepository) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.ProjectCdnConfig, error) {
	var configs []model.ProjectCdnConfig
	err := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
		Order("name").
		Find(&configs).Error
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *projectCdnConfigRepository) FindEnabledByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.ProjectCdnConfig, error) {
	var configs []model.ProjectCdnConfig
	err := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND enabled = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode, true).
		Order("name").
		Find(&configs).Error
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *projectCdnConfigRepository) FindByName(ctx context.Context, namespaceCode, projectCode, name string) (*model.ProjectCdnConfig, error) {
	var config model.ProjectCdnConfig
	err := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND name = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode, name).
		First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *projectCdnConfigRepository) Create(ctx context.Context, config *model.ProjectCdnConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *projectCdnConfigRepository) Update(ctx context.Context, config *model.ProjectCdnConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}

func (r *projectCdnConfigRepository) Delete(ctx context.Context, namespaceCode, projectCode, name string) error {
	result := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND name = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode, name).
		Delete(&model.ProjectCdnConfig{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
