package repository

import (
	"context"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type NamespaceRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, namespace *model.Namespace) error
	Update(ctx context.Context, namespace *model.Namespace) error
	DeleteByCode(ctx context.Context, code string) error
	FindByCode(ctx context.Context, code string) (*model.Namespace, error)
	FindAll(ctx context.Context) ([]model.Namespace, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.Namespace, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Namespace, int64, error)
}

type namespaceRepository struct {
	db *gorm.DB
}

func NewNamespaceRepository(db *gorm.DB) NamespaceRepository {
	return &namespaceRepository{db: db}
}

func (r *namespaceRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *namespaceRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Namespace{})
}

func (r *namespaceRepository) Create(ctx context.Context, namespace *model.Namespace) error {
	return r.db.WithContext(ctx).Create(namespace).Error
}

func (r *namespaceRepository) Update(ctx context.Context, namespace *model.Namespace) error {
	return r.db.WithContext(ctx).Save(namespace).Error
}

func (r *namespaceRepository) DeleteByCode(ctx context.Context, code string) error {
	return r.db.WithContext(ctx).Where("namespace_code = ?", code).Delete(&model.Namespace{}).Error
}

func (r *namespaceRepository) FindByCode(ctx context.Context, code string) (*model.Namespace, error) {
	var namespace model.Namespace
	err := r.db.WithContext(ctx).Where("namespace_code = ?", code).First(&namespace).Error
	if err != nil {
		return nil, err
	}
	return &namespace, nil
}

func (r *namespaceRepository) FindAll(ctx context.Context) ([]model.Namespace, error) {
	var namespaces []model.Namespace
	err := r.db.WithContext(ctx).WithContext(ctx).Find(&namespaces).Error
	return namespaces, err
}

func (r *namespaceRepository) Search(ctx context.Context, query *gorm.DB) ([]model.Namespace, error) {
	namespaces, _, err := r.SearchPaginate(ctx, query, 0, 0)
	return namespaces, err
}

func (r *namespaceRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Namespace, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.Namespace{})
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}
	var namespaces []model.Namespace
	if err := query.Find(&namespaces).Error; err != nil {
		return nil, 0, err
	}

	return namespaces, total, nil
}
