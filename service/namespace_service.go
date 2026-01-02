package service

import (
	"context"

	"github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type NamespaceService interface {
	Create(ctx context.Context, input *model.Namespace) (*model.Namespace, error)
	Update(ctx context.Context, namespaceCode string, input model.Namespace) (*model.Namespace, error)
	Delete(ctx context.Context, namespaceCode string) (bool, error)
	GetByCode(ctx context.Context, namespaceCode string) (*model.Namespace, error)
	GetAll(ctx context.Context) ([]model.Namespace, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.Namespace, error)
	SearchPaginate(ctx context.Context, pagination *types.PaginationInput, query *gorm.DB) (*model.NamespaceList, error)
}

type namespaceService struct {
	validator   *validator.Validate
	repo        repository.NamespaceRepository
	projectRepo repository.ProjectRepository
}

func NewNamespaceService(
	validator *validator.Validate,
	repo repository.NamespaceRepository,
	projectRepo repository.ProjectRepository,
) NamespaceService {
	return &namespaceService{
		validator:   validator,
		repo:        repo,
		projectRepo: projectRepo,
	}
}

func (s *namespaceService) Create(ctx context.Context, input *model.Namespace) (*model.Namespace, error) {

	err := s.validator.Struct(input)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Create(ctx, input); err != nil {
		return nil, err
	}

	return input, nil
}

func (s *namespaceService) Update(ctx context.Context, namespaceCode string, input model.Namespace) (*model.Namespace, error) {
	namespace, err := s.repo.FindByCode(ctx, namespaceCode)
	if err != nil {
		return nil, err
	}

	namespace.Name = input.Name
	err = s.validator.Struct(namespace)
	if err != nil {
		return nil, err
	}

	if err = s.repo.Update(ctx, namespace); err != nil {
		return nil, err
	}

	return namespace, nil
}

func (s *namespaceService) Delete(ctx context.Context, namespaceCode string) (bool, error) {
	// Delete associated projects first
	if err := s.projectRepo.DeleteByNamespaceCode(ctx, namespaceCode); err != nil {
		return false, err
	}

	if err := s.repo.DeleteByCode(ctx, namespaceCode); err != nil {
		return false, err
	}

	return true, nil
}

func (s *namespaceService) GetByCode(ctx context.Context, namespaceCode string) (*model.Namespace, error) {
	return s.repo.FindByCode(ctx, namespaceCode)
}

func (s *namespaceService) GetAll(ctx context.Context) ([]model.Namespace, error) {
	return s.repo.FindAll(ctx)
}

func (s *namespaceService) Search(ctx context.Context, query *gorm.DB) ([]model.Namespace, error) {
	return s.repo.Search(ctx, query)
}

func (s *namespaceService) SearchPaginate(ctx context.Context, pagination *types.PaginationInput, query *gorm.DB) (*model.NamespaceList, error) {
	namespaces, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.NamespaceList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  namespaces,
	}, nil
}
