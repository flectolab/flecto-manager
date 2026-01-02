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

func setupNamespaceServiceTest(t *testing.T) (*gomock.Controller, *mockFlectoRepository.MockNamespaceRepository, *mockFlectoRepository.MockProjectRepository, NamespaceService) {
	ctrl := gomock.NewController(t)
	mockNsRepo := mockFlectoRepository.NewMockNamespaceRepository(ctrl)
	mockProjRepo := mockFlectoRepository.NewMockProjectRepository(ctrl)
	svc := NewNamespaceService(appContext.TestContext(nil), mockNsRepo, mockProjRepo)
	return ctrl, mockNsRepo, mockProjRepo, svc
}

func TestNewNamespaceService(t *testing.T) {
	ctrl, mockNsRepo, mockProjRepo, svc := setupNamespaceServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockNsRepo)
	assert.NotNil(t, mockProjRepo)
}

func TestNamespaceService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.Namespace{
			NamespaceCode: "test-ns",
			Name:          "Test Namespace",
		}

		mockNsRepo.EXPECT().
			Create(ctx, input).
			Return(nil)

		result, err := svc.Create(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		ctrl, _, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.Namespace{
			NamespaceCode: "test ns",
			Name:          "Test Namespace",
		}

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'NamespaceCode' failed on the 'code' tag")
		assert.Nil(t, result)
	})

	t.Run("repository error", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.Namespace{
			NamespaceCode: "test-ns",
			Name:          "Test Namespace",
		}
		expectedErr := errors.New("database error")

		mockNsRepo.EXPECT().
			Create(ctx, input).
			Return(expectedErr)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestNamespaceService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingNs := &model.Namespace{
			ID:            1,
			NamespaceCode: "test-ns",
			Name:          "Original Name",
		}
		input := model.Namespace{
			NamespaceCode: "test-ns",
			Name:          "Updated Name",
		}

		mockNsRepo.EXPECT().
			FindByCode(ctx, "test-ns").
			Return(existingNs, nil)

		mockNsRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, ns *model.Namespace) error {
				assert.Equal(t, "Updated Name", ns.Name)
				return nil
			})

		result, err := svc.Update(ctx, "test-ns", input)

		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", result.Name)
	})

	t.Run("namespace not found", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := model.Namespace{Name: "Updated Name"}
		expectedErr := errors.New("record not found")

		mockNsRepo.EXPECT().
			FindByCode(ctx, "non-existing").
			Return(nil, expectedErr)

		result, err := svc.Update(ctx, "non-existing", input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingNs := &model.Namespace{
			ID:            1,
			NamespaceCode: "test-ns",
			Name:          "Original Name",
		}
		input := model.Namespace{}

		mockNsRepo.EXPECT().
			FindByCode(ctx, "test-ns").
			Return(existingNs, nil)

		result, err := svc.Update(ctx, "test-ns", input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'Name' failed on the 'required' tag")
		assert.Nil(t, result)
	})

	t.Run("update error", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingNs := &model.Namespace{
			ID:            1,
			NamespaceCode: "test-ns",
			Name:          "Original Name",
		}
		input := model.Namespace{Name: "Updated Name"}
		expectedErr := errors.New("update failed")

		mockNsRepo.EXPECT().
			FindByCode(ctx, "test-ns").
			Return(existingNs, nil)

		mockNsRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(expectedErr)

		result, err := svc.Update(ctx, "test-ns", input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestNamespaceService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockNsRepo, mockProjRepo, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockProjRepo.EXPECT().
			DeleteByNamespaceCode(ctx, "test-ns").
			Return(nil)

		mockNsRepo.EXPECT().
			DeleteByCode(ctx, "test-ns").
			Return(nil)

		result, err := svc.Delete(ctx, "test-ns")

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("delete projects error", func(t *testing.T) {
		ctrl, _, mockProjRepo, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("delete projects failed")

		mockProjRepo.EXPECT().
			DeleteByNamespaceCode(ctx, "test-ns").
			Return(expectedErr)

		result, err := svc.Delete(ctx, "test-ns")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})

	t.Run("delete namespace error", func(t *testing.T) {
		ctrl, mockNsRepo, mockProjRepo, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("delete namespace failed")

		mockProjRepo.EXPECT().
			DeleteByNamespaceCode(ctx, "test-ns").
			Return(nil)

		mockNsRepo.EXPECT().
			DeleteByCode(ctx, "test-ns").
			Return(expectedErr)

		result, err := svc.Delete(ctx, "test-ns")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})
}

func TestNamespaceService_GetByCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedNs := &model.Namespace{
			ID:            1,
			NamespaceCode: "test-ns",
			Name:          "Test Namespace",
		}

		mockNsRepo.EXPECT().
			FindByCode(ctx, "test-ns").
			Return(expectedNs, nil)

		result, err := svc.GetByCode(ctx, "test-ns")

		assert.NoError(t, err)
		assert.Equal(t, expectedNs, result)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		mockNsRepo.EXPECT().
			FindByCode(ctx, "non-existing").
			Return(nil, expectedErr)

		result, err := svc.GetByCode(ctx, "non-existing")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestNamespaceService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedNs := []model.Namespace{
			{ID: 1, NamespaceCode: "ns-1", Name: "Namespace 1"},
			{ID: 2, NamespaceCode: "ns-2", Name: "Namespace 2"},
		}

		mockNsRepo.EXPECT().
			FindAll(ctx).
			Return(expectedNs, nil)

		result, err := svc.GetAll(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expectedNs, result)
	})

	t.Run("empty result", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockNsRepo.EXPECT().
			FindAll(ctx).
			Return([]model.Namespace{}, nil)

		result, err := svc.GetAll(ctx)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mockNsRepo.EXPECT().
			FindAll(ctx).
			Return(nil, expectedErr)

		result, err := svc.GetAll(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestNamespaceService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedNs := []model.Namespace{
			{ID: 1, NamespaceCode: "ns-1", Name: "Namespace 1"},
		}

		mockNsRepo.EXPECT().
			Search(ctx, nil).
			Return(expectedNs, nil)

		result, err := svc.Search(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, expectedNs, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("search error")

		mockNsRepo.EXPECT().
			Search(ctx, nil).
			Return(nil, expectedErr)

		result, err := svc.Search(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestNamespaceService_SearchPaginate(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &types.PaginationInput{
			Limit:  &limit,
			Offset: &offset,
		}
		expectedNs := []model.Namespace{
			{ID: 1, NamespaceCode: "ns-1", Name: "Namespace 1"},
			{ID: 2, NamespaceCode: "ns-2", Name: "Namespace 2"},
		}

		mockNsRepo.EXPECT().
			SearchPaginate(ctx, nil, 10, 5).
			Return(expectedNs, int64(20), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 20, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 5, result.Offset)
		assert.Len(t, result.Items, 2)
	})

	t.Run("success with default pagination", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedNs := []model.Namespace{
			{ID: 1, NamespaceCode: "ns-1", Name: "Namespace 1"},
		}

		mockNsRepo.EXPECT().
			SearchPaginate(ctx, nil, types.DefaultLimit, types.DefaultOffset).
			Return(expectedNs, int64(1), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, types.DefaultLimit, result.Limit)
		assert.Equal(t, types.DefaultOffset, result.Offset)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedErr := errors.New("search error")

		mockNsRepo.EXPECT().
			SearchPaginate(ctx, nil, types.DefaultLimit, types.DefaultOffset).
			Return(nil, int64(0), expectedErr)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestNamespaceService_GetTx(t *testing.T) {
	ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockNsRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestNamespaceService_GetQuery(t *testing.T) {
	ctrl, mockNsRepo, _, svc := setupNamespaceServiceTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockNsRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}
