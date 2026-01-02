package repository

import (
	"context"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type RedirectDraftRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	FindByID(ctx context.Context, id int64) (*model.RedirectDraft, error)
	FindByIDWithProject(ctx context.Context, namespaceCode, projectCode string, id int64) (*model.RedirectDraft, error)
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.RedirectDraft, error)
	Create(ctx context.Context, draft *model.RedirectDraft) error
	Update(ctx context.Context, draft *model.RedirectDraft) error
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, query *gorm.DB) ([]model.RedirectDraft, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.RedirectDraft, int64, error)
	CheckSourceAvailability(ctx context.Context, namespaceCode, projectCode, source string, excludeRedirectID, excludeDraftID *int64) (bool, error)
}

type redirectDraftRepository struct {
	db *gorm.DB
}

func NewRedirectDraftRepository(db *gorm.DB) RedirectDraftRepository {
	return &redirectDraftRepository{db: db}
}

func (r *redirectDraftRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *redirectDraftRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.RedirectDraft{})
}

func (r *redirectDraftRepository) FindByID(ctx context.Context, id int64) (*model.RedirectDraft, error) {
	var draft model.RedirectDraft
	err := r.db.WithContext(ctx).
		Preload("OldRedirect").
		Where("id = ?", id).
		First(&draft).Error
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

func (r *redirectDraftRepository) FindByIDWithProject(ctx context.Context, namespaceCode, projectCode string, id int64) (*model.RedirectDraft, error) {
	var draft model.RedirectDraft
	err := r.db.WithContext(ctx).
		Preload("OldRedirect").
		Where("id = ? AND namespace_code = ? AND project_code = ?", id, namespaceCode, projectCode).
		First(&draft).Error
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

func (r *redirectDraftRepository) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.RedirectDraft, error) {
	var drafts []model.RedirectDraft
	err := r.db.WithContext(ctx).
		Preload("OldRedirect").
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Find(&drafts).Error
	if err != nil {
		return nil, err
	}
	return drafts, nil
}

func (r *redirectDraftRepository) Create(ctx context.Context, draft *model.RedirectDraft) error {
	return r.db.WithContext(ctx).Create(draft).Error
}

func (r *redirectDraftRepository) Update(ctx context.Context, draft *model.RedirectDraft) error {
	return r.db.WithContext(ctx).Save(draft).Error
}

func (r *redirectDraftRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.RedirectDraft{}, id).Error
}

func (r *redirectDraftRepository) Search(ctx context.Context, query *gorm.DB) ([]model.RedirectDraft, error) {
	drafts, _, err := r.SearchPaginate(ctx, query, 0, 0)
	return drafts, err
}

func (r *redirectDraftRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.RedirectDraft, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.RedirectDraft{})
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var drafts []model.RedirectDraft
	if err := query.Preload("OldRedirect").Find(&drafts).Error; err != nil {
		return nil, 0, err
	}

	return drafts, total, nil
}

// CheckSourceAvailability checks if a source is available for a project.
// Returns true if available, false if already used.
func (r *redirectDraftRepository) CheckSourceAvailability(ctx context.Context, namespaceCode, projectCode, source string, excludeRedirectID, excludeDraftID *int64) (bool, error) {
	var exists bool

	excludeRedirect := int64(0)
	if excludeRedirectID != nil {
		excludeRedirect = *excludeRedirectID
	}
	excludeDraft := int64(0)
	if excludeDraftID != nil {
		excludeDraft = *excludeDraftID
	}

	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS(
			SELECT 1 FROM redirects
			WHERE namespace_code = ?
			AND project_code = ?
			AND source = ?
			AND id != ?
			UNION
			SELECT 1 FROM redirect_drafts
			WHERE namespace_code = ?
			AND project_code = ?
			AND new_source = ?
			AND id != ?
			AND change_type != 'DELETE'
		)
	`, namespaceCode, projectCode, source, excludeRedirect,
		namespaceCode, projectCode, source, excludeDraft,
	).Scan(&exists).Error

	if err != nil {
		return false, err
	}

	return !exists, nil
}
