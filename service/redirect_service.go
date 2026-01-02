package service

import (
	"context"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"gorm.io/gorm"
)

type RedirectService interface {
	GetByID(ctx context.Context, namespaceCode, projectCode string, redirectID int64) (*model.Redirect, error)
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Redirect, error)
	FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, pagination *commonTypes.PaginationInput) ([]model.Redirect, int64, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.Redirect, error)
	SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.RedirectList, error)
}

type redirectService struct {
	repo repository.RedirectRepository
}

func NewRedirectService(repo repository.RedirectRepository) RedirectService {
	return &redirectService{
		repo: repo,
	}
}

func (s *redirectService) GetByID(ctx context.Context, namespaceCode, projectCode string, redirectID int64) (*model.Redirect, error) {
	return s.repo.FindByID(ctx, namespaceCode, projectCode, redirectID)
}

func (s *redirectService) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Redirect, error) {
	return s.repo.FindByProject(ctx, namespaceCode, projectCode)
}

func (s *redirectService) FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, pagination *commonTypes.PaginationInput) ([]model.Redirect, int64, error) {
	return s.repo.FindByProjectPublished(ctx, namespaceCode, projectCode, pagination.GetLimit(), pagination.GetOffset())
}

func (s *redirectService) Search(ctx context.Context, query *gorm.DB) ([]model.Redirect, error) {
	return s.repo.Search(ctx, query)
}

func (s *redirectService) SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.RedirectList, error) {
	redirects, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.RedirectList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  redirects,
	}, nil
}
