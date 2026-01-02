package repository

import (
	"context"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type ProjectRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, project *model.Project) error
	Update(ctx context.Context, project *model.Project) error
	Delete(ctx context.Context, namespaceCode, projectCode string) error
	DeleteByNamespaceCode(ctx context.Context, namespaceCode string) error
	FindByCode(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error)
	FindByCodeWithNamespace(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error)
	FindAll(ctx context.Context) ([]model.Project, error)
	FindByNamespace(ctx context.Context, namespaceCode string) ([]model.Project, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.Project, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Project, int64, error)
	CountRedirects(ctx context.Context, namespaceCode, projectCode string) (int64, error)
	CountRedirectDrafts(ctx context.Context, namespaceCode, projectCode string) (int64, error)
	CountPages(ctx context.Context, namespaceCode, projectCode string) (int64, error)
	CountPageDrafts(ctx context.Context, namespaceCode, projectCode string) (int64, error)
}

type projectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *projectRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Project{})
}

func (r *projectRepository) Create(ctx context.Context, project *model.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *projectRepository) Update(ctx context.Context, project *model.Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

func (r *projectRepository) Delete(ctx context.Context, namespaceCode, projectCode string) error {
	return r.db.WithContext(ctx).
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Delete(&model.Project{}).Error
}

func (r *projectRepository) DeleteByNamespaceCode(ctx context.Context, namespaceCode string) error {
	return r.db.WithContext(ctx).Where("namespace_code = ?", namespaceCode).Delete(&model.Project{}).Error
}

func (r *projectRepository) FindByCode(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *projectRepository) FindByCodeWithNamespace(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Preload("Namespace").
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *projectRepository) FindAll(ctx context.Context) ([]model.Project, error) {
	var projects []model.Project
	err := r.db.WithContext(ctx).Find(&projects).Error
	return projects, err
}

func (r *projectRepository) FindByNamespace(ctx context.Context, namespaceCode string) ([]model.Project, error) {
	var projects []model.Project
	err := r.db.WithContext(ctx).Where("namespace_code = ?", namespaceCode).Find(&projects).Error
	return projects, err
}

func (r *projectRepository) Search(ctx context.Context, query *gorm.DB) ([]model.Project, error) {
	projects, _, err := r.SearchPaginate(ctx, query, 0, 0)
	return projects, err
}

func (r *projectRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Project, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.Project{})
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var projects []model.Project
	if err := query.Preload("Namespace").Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

func (r *projectRepository) CountRedirects(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Redirect{}).
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Count(&count).Error
	return count, err
}

func (r *projectRepository) CountRedirectDrafts(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.RedirectDraft{}).
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Count(&count).Error
	return count, err
}

func (r *projectRepository) CountPages(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Page{}).
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Count(&count).Error
	return count, err
}

func (r *projectRepository) CountPageDrafts(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PageDraft{}).
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Count(&count).Error
	return count, err
}
