package service

import (
	"context"
	"errors"
	"testing"

	"github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setupPageServiceTest(t *testing.T) (*gomock.Controller, *mockFlectoRepository.MockPageRepository, PageService) {
	ctrl := gomock.NewController(t)
	mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
	svc := NewPageService(appContext.TestContext(nil), mockPageRepo)
	return ctrl, mockPageRepo, svc
}

func TestNewPageService(t *testing.T) {
	ctrl, mockPageRepo, svc := setupPageServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockPageRepo)
}

func TestPageService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedPage := &model.Page{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
		}

		mockPageRepo.EXPECT().
			FindByID(ctx, "test-ns", "test-proj", int64(1)).
			Return(expectedPage, nil)

		result, err := svc.GetByID(ctx, "test-ns", "test-proj", 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedPage, result)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		mockPageRepo.EXPECT().
			FindByID(ctx, "test-ns", "test-proj", int64(999)).
			Return(nil, expectedErr)

		result, err := svc.GetByID(ctx, "test-ns", "test-proj", 999)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestPageService_FindByProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedPages := []model.Page{
			{
				ID:            1,
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
			},
			{
				ID:            2,
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
			},
		}

		mockPageRepo.EXPECT().
			FindByProject(ctx, "test-ns", "test-proj").
			Return(expectedPages, nil)

		result, err := svc.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, expectedPages, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mockPageRepo.EXPECT().
			FindByProject(ctx, "test-ns", "test-proj").
			Return(nil, expectedErr)

		result, err := svc.FindByProject(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestPageService_FindByProjectPublished(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &types.PaginationInput{
			Limit:  &limit,
			Offset: &offset,
		}
		expectedPages := []model.Page{
			{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
			{ID: 2, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
		}

		mockPageRepo.EXPECT().
			FindByProjectPublished(ctx, "test-ns", "test-proj", 10, 5).
			Return(expectedPages, int64(50), nil)

		result, total, err := svc.FindByProjectPublished(ctx, "test-ns", "test-proj", pagination)

		assert.NoError(t, err)
		assert.Equal(t, expectedPages, result)
		assert.Equal(t, int64(50), total)
	})

	t.Run("success with default pagination", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedPages := []model.Page{
			{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
		}

		mockPageRepo.EXPECT().
			FindByProjectPublished(ctx, "test-ns", "test-proj", types.DefaultLimit, types.DefaultOffset).
			Return(expectedPages, int64(1), nil)

		result, total, err := svc.FindByProjectPublished(ctx, "test-ns", "test-proj", pagination)

		assert.NoError(t, err)
		assert.Equal(t, expectedPages, result)
		assert.Equal(t, int64(1), total)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedErr := errors.New("database error")

		mockPageRepo.EXPECT().
			FindByProjectPublished(ctx, "test-ns", "test-proj", types.DefaultLimit, types.DefaultOffset).
			Return(nil, int64(0), expectedErr)

		result, total, err := svc.FindByProjectPublished(ctx, "test-ns", "test-proj", pagination)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
		assert.Equal(t, int64(0), total)
	})

	t.Run("empty result", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}

		mockPageRepo.EXPECT().
			FindByProjectPublished(ctx, "test-ns", "test-proj", types.DefaultLimit, types.DefaultOffset).
			Return([]model.Page{}, int64(0), nil)

		result, total, err := svc.FindByProjectPublished(ctx, "test-ns", "test-proj", pagination)

		assert.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, int64(0), total)
	})
}

func TestPageService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedPages := []model.Page{
			{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
			{ID: 2, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
		}

		mockPageRepo.EXPECT().
			Search(ctx, nil).
			Return(expectedPages, nil)

		result, err := svc.Search(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, expectedPages, result)
	})

	t.Run("empty result", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockPageRepo.EXPECT().
			Search(ctx, nil).
			Return([]model.Page{}, nil)

		result, err := svc.Search(ctx, nil)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("search error")

		mockPageRepo.EXPECT().
			Search(ctx, nil).
			Return(nil, expectedErr)

		result, err := svc.Search(ctx, nil)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestPageService_SearchPaginate(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &types.PaginationInput{
			Limit:  &limit,
			Offset: &offset,
		}
		expectedPages := []model.Page{
			{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
			{ID: 2, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
		}

		mockPageRepo.EXPECT().
			SearchPaginate(ctx, nil, 10, 5).
			Return(expectedPages, int64(50), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 50, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 5, result.Offset)
		assert.Len(t, result.Items, 2)
	})

	t.Run("success with default pagination", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedPages := []model.Page{
			{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
		}

		mockPageRepo.EXPECT().
			SearchPaginate(ctx, nil, types.DefaultLimit, types.DefaultOffset).
			Return(expectedPages, int64(1), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, types.DefaultLimit, result.Limit)
		assert.Equal(t, types.DefaultOffset, result.Offset)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockPageRepo, svc := setupPageServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedErr := errors.New("search error")

		mockPageRepo.EXPECT().
			SearchPaginate(ctx, nil, types.DefaultLimit, types.DefaultOffset).
			Return(nil, int64(0), expectedErr)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}
func TestPageService_GetTx(t *testing.T) {
	ctrl, mockPageRepo, svc := setupPageServiceTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockPageRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestPageService_GetQuery(t *testing.T) {
	ctrl, mockPageRepo, svc := setupPageServiceTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockPageRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}
