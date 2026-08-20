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
	FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, limit, offset int, afterID *int64) ([]model.Redirect, int64, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.Redirect, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Redirect, int64, error)
	SearchBatch(ctx context.Context, query *gorm.DB, batchSize int, fn func([]model.Redirect) error) error
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

func (r *redirectRepository) FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, limit, offset int, afterID *int64) ([]model.Redirect, int64, error) {
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Redirect{}).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND is_published = 1", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode)

	// Cursor pagination walks from a position instead of skipping rows, so it also
	// skips the count: the caller already has the total from the first page. Counting
	// on every page is what makes a full sync of a large project slow.
	if afterID == nil {
		if err := query.Count(&total).Error; err != nil {
			return nil, 0, err
		}
	} else {
		query = query.Where("id > ?", *afterID)
	}

	// ORDER BY on both paths, not just the cursor one: without it the order is
	// whatever plan the optimiser picked, so two pages of the same listing are not
	// guaranteed to line up and a client can skip or repeat rows.
	query = query.Order("id")

	if limit != 0 {
		query = query.Limit(limit)
		if afterID == nil {
			query = query.Offset(offset)
		}
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

func (r *redirectRepository) SearchBatch(ctx context.Context, query *gorm.DB, batchSize int, fn func([]model.Redirect) error) error {
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.Redirect{})
	}

	var redirects []model.Redirect
	result := query.FindInBatches(&redirects, batchSize, func(tx *gorm.DB, batch int) error {
		return fn(redirects)
	})
	return result.Error
}
