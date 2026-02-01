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

var ErrSourceAlreadyUsed = errors.New("source is already used in this project")

type RedirectDraftService interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	GetByID(ctx context.Context, id int64) (*model.RedirectDraft, error)
	GetByIDWithProject(ctx context.Context, namespaceCode, projectCode string, id int64) (*model.RedirectDraft, error)
	Create(ctx context.Context, namespaceCode, projectCode string, oldRedirectID *int64, newRedirect *commonTypes.Redirect) (*model.RedirectDraft, error)
	Update(ctx context.Context, id int64, newRedirect *commonTypes.Redirect) (*model.RedirectDraft, error)
	Delete(ctx context.Context, id int64) (bool, error)
	Rollback(ctx context.Context, namespaceCode, projectCode string) (bool, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.RedirectDraft, error)
	SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.RedirectDraftList, error)
}

type redirectDraftService struct {
	ctx  *appContext.Context
	repo repository.RedirectDraftRepository
}

func NewRedirectDraftService(ctx *appContext.Context, repo repository.RedirectDraftRepository) RedirectDraftService {
	return &redirectDraftService{
		ctx:  ctx,
		repo: repo,
	}
}

func (s *redirectDraftService) GetTx(ctx context.Context) *gorm.DB {
	return s.repo.GetTx(ctx)
}

func (s *redirectDraftService) GetQuery(ctx context.Context) *gorm.DB {
	return s.repo.GetQuery(ctx)
}

func (s *redirectDraftService) GetByID(ctx context.Context, id int64) (*model.RedirectDraft, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *redirectDraftService) GetByIDWithProject(ctx context.Context, namespaceCode, projectCode string, id int64) (*model.RedirectDraft, error) {
	return s.repo.FindByIDWithProject(ctx, namespaceCode, projectCode, id)
}

func (s *redirectDraftService) Create(ctx context.Context, namespaceCode, projectCode string, oldRedirectID *int64, newRedirect *commonTypes.Redirect) (*model.RedirectDraft, error) {
	if oldRedirectID == nil && newRedirect == nil {
		return nil, fmt.Errorf("oldRedirectID or newRedirect must be provided")
	}

	redirectDraft := &model.RedirectDraft{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		ChangeType:    model.DraftChangeTypeCreate,
	}

	if oldRedirectID != nil {
		redirectDraft.OldRedirectID = oldRedirectID
		redirectDraft.ChangeType = model.DraftChangeTypeUpdate
	}

	if newRedirect != nil {
		redirectDraft.NewRedirect = newRedirect

		// Check source availability
		available, err := s.repo.CheckSourceAvailability(ctx, namespaceCode, projectCode, newRedirect.Source, oldRedirectID, nil)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, ErrSourceAlreadyUsed
		}
	} else {
		redirectDraft.ChangeType = model.DraftChangeTypeDelete
	}

	if redirectDraft.ChangeType != model.DraftChangeTypeDelete {
		errValidate := s.ctx.Validator.Struct(redirectDraft.NewRedirect)
		if errValidate != nil {
			return nil, errValidate
		}
	}

	err := s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		if redirectDraft.ChangeType == model.DraftChangeTypeCreate {
			redirect := &model.Redirect{
				NamespaceCode: namespaceCode,
				ProjectCode:   projectCode,
				IsPublished:   types.Ptr(false),
			}
			if err := tx.Create(redirect).Error; err != nil {
				return err
			}
			redirectDraft.OldRedirectID = types.Ptr(redirect.ID)
			redirectDraft.OldRedirect = redirect
		}
		if err := tx.Create(redirectDraft).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reload with preloads
	return s.repo.FindByID(ctx, redirectDraft.ID)
}

func (s *redirectDraftService) Update(ctx context.Context, id int64, newRedirect *commonTypes.Redirect) (*model.RedirectDraft, error) {
	if newRedirect == nil {
		return nil, fmt.Errorf("newRedirect must be provided")
	}

	draft, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if draft.ChangeType == model.DraftChangeTypeDelete {
		return nil, fmt.Errorf("cannot update a delete draft")
	}

	errValidate := s.ctx.Validator.Struct(newRedirect)
	if errValidate != nil {
		return nil, errValidate
	}

	// Check source availability if source changed
	if draft.NewRedirect == nil || draft.NewRedirect.Source != newRedirect.Source {
		available, err := s.repo.CheckSourceAvailability(ctx, draft.NamespaceCode, draft.ProjectCode, newRedirect.Source, draft.OldRedirectID, &draft.ID)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, ErrSourceAlreadyUsed
		}
	}

	draft.NewRedirect = newRedirect

	if err = s.repo.Update(ctx, draft); err != nil {
		return nil, err
	}

	return draft, nil
}

func (s *redirectDraftService) Delete(ctx context.Context, id int64) (bool, error) {
	draft, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return false, err
	}

	err = s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.Delete(&model.RedirectDraft{}, id).Error; err != nil {
			return err
		}
		if draft.ChangeType == model.DraftChangeTypeCreate && draft.OldRedirectID != nil {
			if err = tx.Delete(&model.Redirect{}, *draft.OldRedirectID).Error; err != nil {
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

func (s *redirectDraftService) Rollback(ctx context.Context, namespaceCode, projectCode string) (bool, error) {
	s.ctx.Logger.Info("redirect drafts rollback started", "namespace", namespaceCode, "project", projectCode)

	err := s.repo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
			Delete(&model.RedirectDraft{}).Error; err != nil {
			return err
		}

		if err := tx.Where(fmt.Sprintf("%s = ? AND %s = ? AND is_published = 0", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
			Delete(&model.Redirect{}).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.ctx.Logger.Error("redirect drafts rollback failed", "namespace", namespaceCode, "project", projectCode, "error", err)
		return false, err
	}

	s.ctx.Logger.Info("redirect drafts rollback completed", "namespace", namespaceCode, "project", projectCode)
	return true, nil
}

func (s *redirectDraftService) Search(ctx context.Context, query *gorm.DB) ([]model.RedirectDraft, error) {
	return s.repo.Search(ctx, query)
}

func (s *redirectDraftService) SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.RedirectDraftList, error) {
	drafts, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.RedirectDraftList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  drafts,
	}, nil
}
