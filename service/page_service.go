package service

import (
	"context"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"gorm.io/gorm"
)

type PageService interface {
	GetByID(ctx context.Context, namespaceCode, projectCode string, pageID int64) (*model.Page, error)
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Page, error)
	FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, pagination *commonTypes.PaginationInput) ([]model.Page, int64, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.Page, error)
	SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.PageList, error)
}

type pageService struct {
	repo repository.PageRepository
}

func NewPageService(repo repository.PageRepository) PageService {
	return &pageService{
		repo: repo,
	}
}

func (s *pageService) GetByID(ctx context.Context, namespaceCode, projectCode string, pageID int64) (*model.Page, error) {
	return s.repo.FindByID(ctx, namespaceCode, projectCode, pageID)
}

func (s *pageService) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Page, error) {
	return s.repo.FindByProject(ctx, namespaceCode, projectCode)
}

func (s *pageService) FindByProjectPublished(ctx context.Context, namespaceCode, projectCode string, pagination *commonTypes.PaginationInput) ([]model.Page, int64, error) {
	return s.repo.FindByProjectPublished(ctx, namespaceCode, projectCode, pagination.GetLimit(), pagination.GetOffset())
}

func (s *pageService) Search(ctx context.Context, query *gorm.DB) ([]model.Page, error) {
	return s.repo.Search(ctx, query)
}

func (s *pageService) SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.PageList, error) {
	pages, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.PageList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  pages,
	}, nil
}