package repository

import (
	"context"
	"fmt"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type PageRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	FindByID(ctx context.Context, namespaceCode, projectCode string, pageID int64) (*model.Page, error)
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Page, error)
	FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, limit, offset int) ([]model.Page, int64, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.Page, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Page, int64, error)
	GetTotalContentSize(ctx context.Context, namespaceCode, projectCode string) (int64, error)
}

type pageRepository struct {
	db *gorm.DB
}

func NewPageRepository(db *gorm.DB) PageRepository {
	return &pageRepository{db: db}
}

func (r *pageRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *pageRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Page{})
}

func (r *pageRepository) FindByID(ctx context.Context, namespaceCode, projectCode string, pageID int64) (*model.Page, error) {
	var page model.Page
	err := r.db.WithContext(ctx).
		Preload("PageDraft").
		Where(fmt.Sprintf("id = ? AND %s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), pageID, namespaceCode, projectCode).
		First(&page).Error
	if err != nil {
		return nil, err
	}
	return &page, nil
}

func (r *pageRepository) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Page, error) {
	var pages []model.Page
	err := r.db.WithContext(ctx).
		Preload("PageDraft").
		Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
		Find(&pages).Error
	if err != nil {
		return nil, err
	}
	return pages, nil
}

func (r *pageRepository) FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, limit, offset int) ([]model.Page, int64, error) {
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Page{}).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND is_published = 1", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var pages []model.Page
	if err := query.Find(&pages).Error; err != nil {
		return nil, 0, err
	}

	return pages, total, nil
}

func (r *pageRepository) Search(ctx context.Context, query *gorm.DB) ([]model.Page, error) {
	pages, _, err := r.SearchPaginate(ctx, query, 0, 0)
	return pages, err
}

func (r *pageRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Page, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.Page{})
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var pages []model.Page
	if err := query.Preload("PageDraft").Find(&pages).Error; err != nil {
		return nil, 0, err
	}

	return pages, total, nil
}

// GetTotalContentSize returns the projected total content size for a project.
// It sums:
// - ContentSize of published pages that don't have a pending draft
// - ContentSize of all CREATE/UPDATE drafts (which represent the new sizes)
func (r *pageRepository) GetTotalContentSize(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	var totalSize int64

	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE((
				SELECT SUM(p.content_size)
				FROM pages p
				WHERE p.namespace_code = ?
				AND p.project_code = ?
				AND p.is_published = 1
				AND NOT EXISTS (
					SELECT 1 FROM page_drafts pd
					WHERE pd.old_page_id = p.id
				)
			), 0) +
			COALESCE((
				SELECT SUM(pd.content_size)
				FROM page_drafts pd
				WHERE pd.namespace_code = ?
				AND pd.project_code = ?
				AND pd.change_type IN ('CREATE', 'UPDATE')
			), 0) as total_size
	`, namespaceCode, projectCode, namespaceCode, projectCode).Scan(&totalSize).Error

	if err != nil {
		return 0, err
	}

	return totalSize, nil
}