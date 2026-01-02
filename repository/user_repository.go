package repository

import (
	"context"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindAll(ctx context.Context) ([]model.User, error)
	Search(ctx context.Context, query *gorm.DB) ([]model.User, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.User, int64, error)
	UpdatePassword(ctx context.Context, id int64, hashedPassword string) error
	UpdateStatus(ctx context.Context, id int64, active bool) error
	UpdateRefreshTokenHash(ctx context.Context, id int64, hash string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *userRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.User{})
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.User{}).Error
}

func (r *userRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindAll(ctx context.Context) ([]model.User, error) {
	var users []model.User
	err := r.db.WithContext(ctx).Find(&users).Error
	return users, err
}

func (r *userRepository) Search(ctx context.Context, query *gorm.DB) ([]model.User, error) {
	users, _, err := r.SearchPaginate(ctx, query, 0, 0)
	return users, err
}

func (r *userRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.User, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.User{})
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var users []model.User
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, id int64, hashedPassword string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("password", hashedPassword).Error
}

func (r *userRepository) UpdateStatus(ctx context.Context, id int64, active bool) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("active", active).Error
}

func (r *userRepository) UpdateRefreshTokenHash(ctx context.Context, id int64, hash string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("refresh_token_hash", hash).Error
}