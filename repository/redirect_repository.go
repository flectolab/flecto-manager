package repository

import (
	"context"
	"fmt"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type RedirectRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	FindByID(ctx context.Context, namespaceCode, projectCode string, redirectID int64) (*model.Redirect, error)
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Redirect, error)
	FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, limit, offset int) ([]model.Redirect, int64, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.Redirect, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Redirect, int64, error)
}

type redirectRepository struct {
	db *gorm.DB
}

func NewRedirectRepository(db *gorm.DB) RedirectRepository {
	return &redirectRepository{db: db}
}

func (r *redirectRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *redirectRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Redirect{})
}

func (r *redirectRepository) FindByID(ctx context.Context, namespaceCode, projectCode string, redirectID int64) (*model.Redirect, error) {
	var redirect model.Redirect
	err := r.db.WithContext(ctx).
		Preload("RedirectDraft").
		Where(fmt.Sprintf("id = ? AND %s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), redirectID, namespaceCode, projectCode).
		First(&redirect).Error
	if err != nil {
		return nil, err
	}
	return &redirect, nil
}

func (r *redirectRepository) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Redirect, error) {
	var redirects []model.Redirect
	err := r.db.WithContext(ctx).
		Preload("RedirectDraft").
		Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
		Find(&redirects).Error
	if err != nil {
		return nil, err
	}
	return redirects, nil
}

func (r *redirectRepository) FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, limit, offset int) ([]model.Redirect, int64, error) {
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Redirect{}).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND is_published = 1", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var redirects []model.Redirect
	if err := query.Find(&redirects).Error; err != nil {
		return nil, 0, err
	}

	return redirects, total, nil
}

func (r *redirectRepository) Search(ctx context.Context, query *gorm.DB) ([]model.Redirect, error) {
	redirects, _, err := r.SearchPaginate(ctx, query, 0, 0)
	return redirects, err
}

func (r *redirectRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Redirect, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.Redirect{})
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var redirects []model.Redirect
	if err := query.Preload("RedirectDraft").Find(&redirects).Error; err != nil {
		return nil, 0, err
	}

	return redirects, total, nil
}
