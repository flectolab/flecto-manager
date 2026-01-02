package service

import (
	"context"
	"fmt"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/types"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type ProjectService interface {
	Create(ctx context.Context, input *model.Project) (*model.Project, error)
	Update(ctx context.Context, namespaceCode, projectCode string, input model.Project) (*model.Project, error)
	Delete(ctx context.Context, namespaceCode, projectCode string) (bool, error)
	GetByCode(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error)
	GetByCodeWithNamespace(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error)
	GetByNamespace(ctx context.Context, namespaceCode string) ([]model.Project, error)
	GetAll(ctx context.Context) ([]model.Project, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.Project, error)
	SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.ProjectList, error)
	CountRedirects(ctx context.Context, namespaceCode, projectCode string) (int64, error)
	CountRedirectDrafts(ctx context.Context, namespaceCode, projectCode string) (int64, error)
	CountPages(ctx context.Context, namespaceCode, projectCode string) (int64, error)
	CountPageDrafts(ctx context.Context, namespaceCode, projectCode string) (int64, error)
	TotalPageContentSize(ctx context.Context, namespaceCode, projectCode string) (int64, error)
	TotalPageContentSizeLimit() int64
	Publish(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error)
}

type projectService struct {
	validator         *validator.Validate
	repo              repository.ProjectRepository
	pageRepo          repository.PageRepository
	db                *gorm.DB
	repoRedirectDraft repository.RedirectDraftRepository
	repoPageDraft     repository.PageDraftRepository
	pageConfig        config.PageConfig
}

func NewProjectService(
	validator *validator.Validate,
	repo repository.ProjectRepository,
	pageRepo repository.PageRepository,
	repoRedirectDraft repository.RedirectDraftRepository,
	repoPageDraft repository.PageDraftRepository,
	pageConfig config.PageConfig,
	db *gorm.DB,
) ProjectService {
	return &projectService{
		validator:         validator,
		repo:              repo,
		pageRepo:          pageRepo,
		repoRedirectDraft: repoRedirectDraft,
		repoPageDraft:     repoPageDraft,
		db:                db,
		pageConfig:        pageConfig,
	}
}

func (s *projectService) Create(ctx context.Context, input *model.Project) (*model.Project, error) {
	err := s.validator.Struct(input)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Create(ctx, input); err != nil {
		return nil, err
	}
	return input, nil
}

func (s *projectService) Update(ctx context.Context, namespaceCode, projectCode string, input model.Project) (*model.Project, error) {
	project, err := s.repo.FindByCode(ctx, namespaceCode, projectCode)
	if err != nil {
		return nil, err
	}

	project.Name = input.Name
	err = s.validator.Struct(project)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *projectService) Delete(ctx context.Context, namespaceCode, projectCode string) (bool, error) {
	if err := s.repo.Delete(ctx, namespaceCode, projectCode); err != nil {
		return false, err
	}
	return true, nil
}

func (s *projectService) GetByCode(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error) {
	return s.repo.FindByCode(ctx, namespaceCode, projectCode)
}

func (s *projectService) GetByCodeWithNamespace(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error) {
	return s.repo.FindByCodeWithNamespace(ctx, namespaceCode, projectCode)
}

func (s *projectService) GetByNamespace(ctx context.Context, namespaceCode string) ([]model.Project, error) {
	return s.repo.FindByNamespace(ctx, namespaceCode)
}

func (s *projectService) GetAll(ctx context.Context) ([]model.Project, error) {
	return s.repo.FindAll(ctx)
}

func (s *projectService) Search(ctx context.Context, query *gorm.DB) ([]model.Project, error) {
	return s.repo.Search(ctx, query)
}
func (s *projectService) SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.ProjectList, error) {
	projects, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.ProjectList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  projects,
	}, nil
}

func (s *projectService) CountRedirects(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	return s.repo.CountRedirects(ctx, namespaceCode, projectCode)
}

func (s *projectService) CountRedirectDrafts(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	return s.repo.CountRedirectDrafts(ctx, namespaceCode, projectCode)
}

func (s *projectService) CountPages(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	return s.repo.CountPages(ctx, namespaceCode, projectCode)
}

func (s *projectService) CountPageDrafts(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	return s.repo.CountPageDrafts(ctx, namespaceCode, projectCode)
}

func (s *projectService) TotalPageContentSize(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	return s.pageRepo.GetTotalContentSize(ctx, namespaceCode, projectCode)
}

func (s *projectService) TotalPageContentSizeLimit() int64 {
	return int64(s.pageConfig.TotalSizeLimit)
}

func (s *projectService) Publish(ctx context.Context, namespaceCode, projectCode string) (*model.Project, error) {
	project, err := s.repo.FindByCode(ctx, namespaceCode, projectCode)
	if err != nil {
		return nil, err
	}

	redirectDraftCount, errRedirectCount := s.CountRedirectDrafts(ctx, namespaceCode, projectCode)
	if errRedirectCount != nil {
		return nil, errRedirectCount
	}
	pageDraftCount, errPageCount := s.CountPageDrafts(ctx, namespaceCode, projectCode)
	if errPageCount != nil {
		return nil, errPageCount
	}

	if redirectDraftCount == 0 && pageDraftCount == 0 {
		return nil, fmt.Errorf("nothing to publish for project %s/%s", namespaceCode, projectCode)
	}

	// Prepare redirect drafts
	redirectDrafts, errGetRedirectDraft := s.repoRedirectDraft.FindByProject(ctx, namespaceCode, projectCode)
	if errGetRedirectDraft != nil {
		return nil, errGetRedirectDraft
	}

	redirects := make([]*model.Redirect, 0)
	redirectsToDelete := make([]int64, 0)
	for _, draft := range redirectDrafts {
		switch draft.ChangeType {
		case model.DraftChangeTypeCreate, model.DraftChangeTypeUpdate:
			redirects = append(redirects, &model.Redirect{ID: *draft.OldRedirectID, IsPublished: types.Ptr(true), NamespaceCode: namespaceCode, ProjectCode: projectCode, Redirect: draft.NewRedirect})
		case model.DraftChangeTypeDelete:
			redirectsToDelete = append(redirectsToDelete, *draft.OldRedirectID)
		}
	}

	// Prepare page drafts
	pageDrafts, errGetPageDraft := s.repoPageDraft.FindByProject(ctx, namespaceCode, projectCode)
	if errGetPageDraft != nil {
		return nil, errGetPageDraft
	}

	pages := make([]*model.Page, 0)
	pagesToDelete := make([]int64, 0)
	for _, draft := range pageDrafts {
		switch draft.ChangeType {
		case model.DraftChangeTypeCreate, model.DraftChangeTypeUpdate:
			pages = append(pages, &model.Page{ID: *draft.OldPageID, IsPublished: types.Ptr(true), NamespaceCode: namespaceCode, ProjectCode: projectCode, ContentSize: draft.ContentSize, Page: draft.NewPage})
		case model.DraftChangeTypeDelete:
			pagesToDelete = append(pagesToDelete, *draft.OldPageID)
		}
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		batchSize := 500

		// Save redirects
		for i := 0; i < len(redirects); i += batchSize {
			end := i + batchSize
			if end > len(redirects) {
				end = len(redirects)
			}

			if err = tx.Save(redirects[i:end]).Error; err != nil {
				return err
			}
		}

		// Delete redirect drafts
		if len(redirectDrafts) > 0 {
			err = tx.Delete(redirectDrafts).Error
			if err != nil {
				return err
			}
		}

		// Delete redirects marked for deletion
		if len(redirectsToDelete) > 0 {
			err = tx.Where("id in ?", redirectsToDelete).Delete(&model.Redirect{}).Error
			if err != nil {
				return err
			}
		}

		// Save pages
		for i := 0; i < len(pages); i += batchSize {
			end := i + batchSize
			if end > len(pages) {
				end = len(pages)
			}

			if err = tx.Save(pages[i:end]).Error; err != nil {
				return err
			}
		}

		// Delete page drafts
		if len(pageDrafts) > 0 {
			err = tx.Delete(pageDrafts).Error
			if err != nil {
				return err
			}
		}

		// Delete pages marked for deletion
		if len(pagesToDelete) > 0 {
			err = tx.Where("id in ?", pagesToDelete).Delete(&model.Page{}).Error
			if err != nil {
				return err
			}
		}

		project.Version++
		err = tx.Save(project).Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return project, nil
}
