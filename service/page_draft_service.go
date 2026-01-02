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
}

func NewPageDraftService(
	ctx *appContext.Context,
	repo repository.PageDraftRepository,
	pageRepo repository.PageRepository,
) PageDraftService {
	return &pageDraftService{
		ctx:      ctx,
		repo:     repo,
		pageRepo: pageRepo,
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

	err := s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
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
		}
		if err := tx.Create(pageDraft).Error; err != nil {
			return err
		}
		return nil
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

	draft.NewPage = newPage
	draft.ContentSize = contentSize

	if err = s.repo.Update(ctx, draft); err != nil {
		return nil, err
	}

	return draft, nil
}

func (s *pageDraftService) Delete(ctx context.Context, id int64) (bool, error) {
	draft, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return false, err
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
		return nil
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *pageDraftService) Rollback(ctx context.Context, namespaceCode, projectCode string) (bool, error) {
	err := s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
			Delete(&model.PageDraft{}).Error; err != nil {
			return err
		}

		if err := tx.Where(fmt.Sprintf("%s = ? AND %s = ? AND is_published = 0", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
			Delete(&model.Page{}).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return false, err
	}

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
