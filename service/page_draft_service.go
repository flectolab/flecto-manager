package service

import (
	"context"
	"errors"
	"fmt"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/types"
	"gorm.io/gorm"
)

var (
	ErrPathAlreadyUsed       = errors.New("path is already used in this project")
	ErrContentSizeExceeded   = errors.New("content size exceeds the maximum allowed size")
	ErrTotalSizeLimitReached = errors.New("total content size limit for the project would be exceeded")
)

type PageDraftService interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	GetByID(ctx context.Context, id int64) (*model.PageDraft, error)
	GetByIDWithProject(ctx context.Context, namespaceCode, projectCode string, id int64) (*model.PageDraft, error)
	Create(ctx context.Context, namespaceCode, projectCode string, oldPageID *int64, newPage *commonTypes.Page) (*model.PageDraft, error)
	Update(ctx context.Context, id int64, newPage *commonTypes.Page) (*model.PageDraft, error)
	Delete(ctx context.Context, id int64) (bool, error)
	Rollback(ctx context.Context, namespaceCode, projectCode string) (bool, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.PageDraft, error)
	SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.PageDraftList, error)
}

type pageDraftService struct {
	ctx      *appContext.Context
	repo     repository.PageDraftRepository
	pageRepo repository.PageRepository
	activity ActivityService
}

func NewPageDraftService(
	ctx *appContext.Context,
	repo repository.PageDraftRepository,
	pageRepo repository.PageRepository,
	activity ActivityService,
) PageDraftService {
	return &pageDraftService{
		ctx:      ctx,
		repo:     repo,
		pageRepo: pageRepo,
		activity: activity,
	}
}

func (s *pageDraftService) GetTx(ctx context.Context) *gorm.DB {
	return s.repo.GetTx(ctx)
}

func (s *pageDraftService) GetQuery(ctx context.Context) *gorm.DB {
	return s.repo.GetQuery(ctx)
}

func (s *pageDraftService) GetByID(ctx context.Context, id int64) (*model.PageDraft, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *pageDraftService) GetByIDWithProject(ctx context.Context, namespaceCode, projectCode string, id int64) (*model.PageDraft, error) {
	return s.repo.FindByIDWithProject(ctx, namespaceCode, projectCode, id)
}

func (s *pageDraftService) Create(ctx context.Context, namespaceCode, projectCode string, oldPageID *int64, newPage *commonTypes.Page) (*model.PageDraft, error) {
	if oldPageID == nil && newPage == nil {
		return nil, fmt.Errorf("oldPageID or newPage must be provided")
	}

	pageDraft := &model.PageDraft{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		ChangeType:    model.DraftChangeTypeCreate,
	}

	if oldPageID != nil {
		pageDraft.OldPageID = oldPageID
		pageDraft.ChangeType = model.DraftChangeTypeUpdate
	}

	if newPage != nil {
		pageDraft.NewPage = newPage
		contentSize := int64(len(newPage.Content))
		pageDraft.ContentSize = contentSize

		// Check content size limit
		if contentSize > int64(s.ctx.Config.Page.SizeLimit) {
			return nil, ErrContentSizeExceeded
		}

		// Check path availability
		available, err := s.repo.CheckPathAvailability(ctx, namespaceCode, projectCode, newPage.Path, oldPageID, nil)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, ErrPathAlreadyUsed
		}

		// Check total size limit
		if err := s.checkTotalSizeLimit(ctx, namespaceCode, projectCode, contentSize); err != nil {
			return nil, err
		}
	} else {
		pageDraft.ChangeType = model.DraftChangeTypeDelete
	}

	if pageDraft.ChangeType != model.DraftChangeTypeDelete {
		errValidate := s.ctx.Validator.Struct(pageDraft.NewPage)
		if errValidate != nil {
			return nil, errValidate
		}
	}

	// The recorded action is what the user did, which the draft change type carries:
	// a draft targeting an existing page is an update, one without a new version is
	// a deletion.
	activityAction := model.ActivityActionCreate
	switch pageDraft.ChangeType {
	case model.DraftChangeTypeUpdate:
		activityAction = model.ActivityActionUpdate
	case model.DraftChangeTypeDelete:
		activityAction = model.ActivityActionDelete
	}

	err := s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		var before *model.PageSnapshot

		if pageDraft.ChangeType == model.DraftChangeTypeCreate {
			page := &model.Page{
				NamespaceCode: namespaceCode,
				ProjectCode:   projectCode,
				IsPublished:   types.Ptr(false),
			}
			if err := tx.Create(page).Error; err != nil {
				return err
			}
			pageDraft.OldPageID = types.Ptr(page.ID)
			pageDraft.OldPage = page
		} else {
			var errBefore error
			if before, errBefore = loadPageSnapshot(tx, *pageDraft.OldPageID); errBefore != nil {
				return errBefore
			}
		}

		if err := tx.Create(pageDraft).Error; err != nil {
			return err
		}
		return s.recordChange(ctx, tx, pageDraft, activityAction, before)
	})
	if err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, pageDraft.ID)
}

func (s *pageDraftService) Update(ctx context.Context, id int64, newPage *commonTypes.Page) (*model.PageDraft, error) {
	if newPage == nil {
		return nil, fmt.Errorf("newPage must be provided")
	}

	draft, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if draft.ChangeType == model.DraftChangeTypeDelete {
		return nil, fmt.Errorf("cannot update a delete draft")
	}

	errValidate := s.ctx.Validator.Struct(newPage)
	if errValidate != nil {
		return nil, errValidate
	}

	contentSize := int64(len(newPage.Content))

	// Check content size limit
	if contentSize > int64(s.ctx.Config.Page.SizeLimit) {
		return nil, ErrContentSizeExceeded
	}

	// Check path availability if path changed
	if draft.NewPage == nil || draft.NewPage.Path != newPage.Path {
		available, err := s.repo.CheckPathAvailability(ctx, draft.NamespaceCode, draft.ProjectCode, newPage.Path, draft.OldPageID, &draft.ID)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, ErrPathAlreadyUsed
		}
	}

	// Check total size limit if content size increased
	oldContentSize := draft.ContentSize
	if contentSize > oldContentSize {
		sizeDiff := contentSize - oldContentSize
		if err := s.checkTotalSizeLimitDiff(ctx, draft.NamespaceCode, draft.ProjectCode, sizeDiff); err != nil {
			return nil, err
		}
	}

	// Editing a pending change is an update whatever the draft change type is:
	// even a pending creation already existed as far as the journal is concerned.
	before := model.NewPageSnapshot(draft.NewPage, draft.ContentSize)

	draft.NewPage = newPage
	draft.ContentSize = contentSize

	err = s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		if errSave := tx.Save(draft).Error; errSave != nil {
			return errSave
		}
		return s.recordChange(ctx, tx, draft, model.ActivityActionUpdate, before)
	})

	if err != nil {
		return nil, err
	}

	return draft, nil
}

// loadPageSnapshot reads the activity projection of a page without its content: a
// page can weigh up to page.size_limit, which must never be pulled into memory
// just to write a journal entry.
func loadPageSnapshot(tx *gorm.DB, pageID int64) (*model.PageSnapshot, error) {
	var page model.Page
	err := tx.Select("id", "type", "path", "content_type", "content_size").
		First(&page, pageID).Error
	if err != nil {
		return nil, err
	}
	return model.NewPageSnapshot(page.Page, page.ContentSize), nil
}

// recordChange records a pending change on a single page. before is the state the
// change replaces, nil when there is none.
func (s *pageDraftService) recordChange(
	ctx context.Context,
	tx *gorm.DB,
	draft *model.PageDraft,
	action model.ActivityAction,
	before *model.PageSnapshot,
) error {
	return s.activity.Record(ctx, tx, model.ActivityInput{
		NamespaceCode: draft.NamespaceCode,
		ProjectCode:   draft.ProjectCode,
		Resource:      model.ActivityResourcePage,
		Action:        action,
		ResourceID:    draft.OldPageID,
		Data: model.ActivityChange[model.PageSnapshot]{
			Before: before,
			After:  model.NewPageSnapshot(draft.NewPage, draft.ContentSize),
		},
	})
}

func (s *pageDraftService) Delete(ctx context.Context, id int64) (bool, error) {
	draft, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return false, err
	}

	// Discarding a draft cancels a pending change: a rollback limited to one entry.
	entry := model.NewPageSnapshot(draft.NewPage, draft.ContentSize)
	if entry == nil && draft.OldPage != nil {
		// A delete draft has no new version, show the page it was going to remove.
		entry = model.NewPageSnapshot(draft.OldPage.Page, draft.OldPage.ContentSize)
	}

	err = s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.Delete(&model.PageDraft{}, id).Error; err != nil {
			return err
		}
		if draft.ChangeType == model.DraftChangeTypeCreate && draft.OldPageID != nil {
			if err = tx.Delete(&model.Page{}, *draft.OldPageID).Error; err != nil {
				return err
			}
		}
		return s.activity.Record(ctx, tx, model.ActivityInput{
			NamespaceCode: draft.NamespaceCode,
			ProjectCode:   draft.ProjectCode,
			Resource:      model.ActivityResourcePage,
			Action:        model.ActivityActionRollback,
			ResourceID:    draft.OldPageID,
			Data: model.ActivityRollback[model.PageSnapshot]{
				Scope:      model.ActivityRollbackScopeSingle,
				ChangeType: &draft.ChangeType,
				Entry:      entry,
			},
		})
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *pageDraftService) Rollback(ctx context.Context, namespaceCode, projectCode string) (bool, error) {
	s.ctx.Logger.Info("page drafts rollback started", "namespace", namespaceCode, "project", projectCode)

	err := s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		// Count before deleting, that is the only thing the activity event can report
		// about a bulk discard.
		discarded, errCount := countDraftsByChangeType(tx, &model.PageDraft{}, namespaceCode, projectCode)
		if errCount != nil {
			return errCount
		}

		if err := tx.Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
			Delete(&model.PageDraft{}).Error; err != nil {
			return err
		}

		if err := tx.Where(fmt.Sprintf("%s = ? AND %s = ? AND is_published = 0", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
			Delete(&model.Page{}).Error; err != nil {
			return err
		}

		// A rollback with nothing pending is a no-op, not worth a journal entry.
		if discarded.IsEmpty() {
			return nil
		}

		return s.activity.Record(ctx, tx, model.ActivityInput{
			NamespaceCode: namespaceCode,
			ProjectCode:   projectCode,
			Resource:      model.ActivityResourcePage,
			Action:        model.ActivityActionRollback,
			Data: model.ActivityRollback[model.PageSnapshot]{
				Scope:     model.ActivityRollbackScopeProject,
				Discarded: discarded,
			},
		})
	})
	if err != nil {
		s.ctx.Logger.Error("page drafts rollback failed", "namespace", namespaceCode, "project", projectCode, "error", err)
		return false, err
	}

	s.ctx.Logger.Info("page drafts rollback completed", "namespace", namespaceCode, "project", projectCode)
	return true, nil
}

func (s *pageDraftService) Search(ctx context.Context, query *gorm.DB) ([]model.PageDraft, error) {
	return s.repo.Search(ctx, query)
}

func (s *pageDraftService) SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.PageDraftList, error) {
	drafts, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.PageDraftList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  drafts,
	}, nil
}

// checkTotalSizeLimit checks if adding a new page with the given content size would exceed the total limit
func (s *pageDraftService) checkTotalSizeLimit(ctx context.Context, namespaceCode, projectCode string, newContentSize int64) error {
	currentTotal, err := s.pageRepo.GetTotalContentSize(ctx, namespaceCode, projectCode)
	if err != nil {
		return err
	}

	if currentTotal+newContentSize > int64(s.ctx.Config.Page.TotalSizeLimit) {
		return ErrTotalSizeLimitReached
	}

	return nil
}

// checkTotalSizeLimitDiff checks if a size difference would exceed the total limit
func (s *pageDraftService) checkTotalSizeLimitDiff(ctx context.Context, namespaceCode, projectCode string, sizeDiff int64) error {
	currentTotal, err := s.pageRepo.GetTotalContentSize(ctx, namespaceCode, projectCode)
	if err != nil {
		return err
	}

	if currentTotal+sizeDiff > int64(s.ctx.Config.Page.TotalSizeLimit) {
		return ErrTotalSizeLimitReached
	}

	return nil
}
