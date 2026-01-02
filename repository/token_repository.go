package repository

import (
	"context"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type TokenRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, token *model.Token) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*model.Token, error)
	FindByName(ctx context.Context, name string) (*model.Token, error)
	FindByHash(ctx context.Context, hash string) (*model.Token, error)
	FindAll(ctx context.Context) ([]model.Token, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Token, int64, error)
}

type tokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *tokenRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Token{})
}

func (r *tokenRepository) Create(ctx context.Context, token *model.Token) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *tokenRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Token{}).Error
}

func (r *tokenRepository) FindByID(ctx context.Context, id int64) (*model.Token, error) {
	var token model.Token
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *tokenRepository) FindByName(ctx context.Context, name string) (*model.Token, error) {
	var token model.Token
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *tokenRepository) FindByHash(ctx context.Context, hash string) (*model.Token, error) {
	var token model.Token
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *tokenRepository) FindAll(ctx context.Context) ([]model.Token, error) {
	var tokens []model.Token
	err := r.db.WithContext(ctx).Find(&tokens).Error
	return tokens, err
}

func (r *tokenRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Token, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.Token{})
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var tokens []model.Token
	if err := query.Find(&tokens).Error; err != nil {
		return nil, 0, err
	}

	return tokens, total, nil
}
