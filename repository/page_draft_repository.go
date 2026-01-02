package repository

import (
	"context"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type PageDraftRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	FindByID(ctx context.Context, id int64) (*model.PageDraft, error)
	FindByIDWithProject(ctx context.Context, namespaceCode, projectCode string, id int64) (*model.PageDraft, error)
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.PageDraft, error)
	Create(ctx context.Context, draft *model.PageDraft) error
	Update(ctx context.Context, draft *model.PageDraft) error
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, query *gorm.DB) ([]model.PageDraft, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.PageDraft, int64, error)
	CheckPathAvailability(ctx context.Context, namespaceCode, projectCode, path string, excludePageID, excludeDraftID *int64) (bool, error)
}

type pageDraftRepository struct {
	db *gorm.DB
}

func NewPageDraftRepository(db *gorm.DB) PageDraftRepository {
	return &pageDraftRepository{db: db}
}

func (r *pageDraftRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *pageDraftRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.PageDraft{})
}

func (r *pageDraftRepository) FindByID(ctx context.Context, id int64) (*model.PageDraft, error) {
	var draft model.PageDraft
	err := r.db.WithContext(ctx).
		Preload("OldPage").
		Where("id = ?", id).
		First(&draft).Error
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

func (r *pageDraftRepository) FindByIDWithProject(ctx context.Context, namespaceCode, projectCode string, id int64) (*model.PageDraft, error) {
	var draft model.PageDraft
	err := r.db.WithContext(ctx).
		Preload("OldPage").
		Where("id = ? AND namespace_code = ? AND project_code = ?", id, namespaceCode, projectCode).
		First(&draft).Error
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

func (r *pageDraftRepository) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.PageDraft, error) {
	var drafts []model.PageDraft
	err := r.db.WithContext(ctx).
		Preload("OldPage").
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Find(&drafts).Error
	if err != nil {
		return nil, err
	}
	return drafts, nil
}

func (r *pageDraftRepository) Create(ctx context.Context, draft *model.PageDraft) error {
	return r.db.WithContext(ctx).Create(draft).Error
}

func (r *pageDraftRepository) Update(ctx context.Context, draft *model.PageDraft) error {
	return r.db.WithContext(ctx).Save(draft).Error
}

func (r *pageDraftRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.PageDraft{}, id).Error
}

func (r *pageDraftRepository) Search(ctx context.Context, query *gorm.DB) ([]model.PageDraft, error) {
	drafts, _, err := r.SearchPaginate(ctx, query, 0, 0)
	return drafts, err
}

func (r *pageDraftRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.PageDraft, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.PageDraft{})
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var drafts []model.PageDraft
	if err := query.Preload("OldPage").Find(&drafts).Error; err != nil {
		return nil, 0, err
	}

	return drafts, total, nil
}

// CheckPathAvailability checks if a path is available for a project.
// Returns true if available, false if already used.
func (r *pageDraftRepository) CheckPathAvailability(ctx context.Context, namespaceCode, projectCode, path string, excludePageID, excludeDraftID *int64) (bool, error) {
	var exists bool

	excludePage := int64(0)
	if excludePageID != nil {
		excludePage = *excludePageID
	}
	excludeDraft := int64(0)
	if excludeDraftID != nil {
		excludeDraft = *excludeDraftID
	}

	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS(
			SELECT 1 FROM pages
			WHERE namespace_code = ?
			AND project_code = ?
			AND path = ?
			AND id != ?
			UNION
			SELECT 1 FROM page_drafts
			WHERE namespace_code = ?
			AND project_code = ?
			AND new_path = ?
			AND id != ?
			AND change_type != 'DELETE'
		)
	`, namespaceCode, projectCode, path, excludePage,
		namespaceCode, projectCode, path, excludeDraft,
	).Scan(&exists).Error

	if err != nil {
		return false, err
	}

	return !exists, nil
}