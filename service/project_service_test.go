package service

import (
	"context"
	"errors"
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/config"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	types "github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var defaultProjectCfg = config.PageConfig{
	SizeLimit:      1024,
	TotalSizeLimit: 2048,
}

type projectServiceTestDeps struct {
	ctrl              *gomock.Controller
	mockProjRepo      *mockFlectoRepository.MockProjectRepository
	mockPageRepo      *mockFlectoRepository.MockPageRepository
	mockRedirectDraft *mockFlectoRepository.MockRedirectDraftRepository
	mockPageDraft     *mockFlectoRepository.MockPageDraftRepository
	svc               ProjectService
}

func setupProjectServiceTest(t *testing.T) *projectServiceTestDeps {
	ctrl := gomock.NewController(t)
	mockProjRepo := mockFlectoRepository.NewMockProjectRepository(ctrl)
	mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
	mockRedirectDraftRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
	mockPageDraftRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
	svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), mockProjRepo, mockPageRepo, mockRedirectDraftRepo, mockPageDraftRepo, newTestActivityService(t, nil))
	return &projectServiceTestDeps{
		ctrl:              ctrl,
		mockProjRepo:      mockProjRepo,
		mockPageRepo:      mockPageRepo,
		mockRedirectDraft: mockRedirectDraftRepo,
		mockPageDraft:     mockPageDraftRepo,
		svc:               svc,
	}
}

func TestNewProjectService(t *testing.T) {
	deps := setupProjectServiceTest(t)
	defer deps.ctrl.Finish()

	assert.NotNil(t, deps.svc)
	assert.NotNil(t, deps.mockProjRepo)
}

func TestProjectService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		input := &model.Project{
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
		}

		deps.mockProjRepo.EXPECT().
			Create(ctx, input).
			Return(nil)

		result, err := deps.svc.Create(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		input := &model.Project{
			ProjectCode:   "test proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
		}

		result, err := deps.svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'ProjectCode' failed on the 'code' tag")
		assert.Nil(t, result)
	})

	t.Run("repository error", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		input := &model.Project{
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
		}
		expectedErr := errors.New("database error")

		deps.mockProjRepo.EXPECT().
			Create(ctx, input).
			Return(expectedErr)

		result, err := deps.svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestProjectService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		existingProj := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Original Name",
		}
		input := model.Project{
			Name: "Updated Name",
		}

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "test-proj").
			Return(existingProj, nil)

		deps.mockProjRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, proj *model.Project) error {
				assert.Equal(t, "Updated Name", proj.Name)
				return nil
			})

		result, err := deps.svc.Update(ctx, "test-ns", "test-proj", input)

		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", result.Name)
	})

	t.Run("project not found", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		input := model.Project{Name: "Updated Name"}
		expectedErr := errors.New("record not found")

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "non-existing").
			Return(nil, expectedErr)

		result, err := deps.svc.Update(ctx, "test-ns", "non-existing", input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		existingProj := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Original Name",
		}
		input := model.Project{}

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "test-proj").
			Return(existingProj, nil)

		result, err := deps.svc.Update(ctx, "test-ns", "test-proj", input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'Name' failed on the 'required' tag")
		assert.Nil(t, result)
	})

	t.Run("update error", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		existingProj := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Original Name",
		}
		input := model.Project{Name: "Updated Name"}
		expectedErr := errors.New("update failed")

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "test-proj").
			Return(existingProj, nil)

		deps.mockProjRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(expectedErr)

		result, err := deps.svc.Update(ctx, "test-ns", "test-proj", input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestProjectService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			Delete(ctx, "test-ns", "test-proj").
			Return(nil)

		result, err := deps.svc.Delete(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("error", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("delete failed")

		deps.mockProjRepo.EXPECT().
			Delete(ctx, "test-ns", "test-proj").
			Return(expectedErr)

		result, err := deps.svc.Delete(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})
}

func TestProjectService_GetByCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedProj := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
		}

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "test-proj").
			Return(expectedProj, nil)

		result, err := deps.svc.GetByCode(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, expectedProj, result)
	})

	t.Run("not found", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "non-existing").
			Return(nil, expectedErr)

		result, err := deps.svc.GetByCode(ctx, "test-ns", "non-existing")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestProjectService_GetByCodeWithNamespace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedProj := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
			Namespace: &model.Namespace{
				ID:            1,
				NamespaceCode: "test-ns",
				Name:          "Test Namespace",
			},
		}

		deps.mockProjRepo.EXPECT().
			FindByCodeWithNamespace(ctx, "test-ns", "test-proj").
			Return(expectedProj, nil)

		result, err := deps.svc.GetByCodeWithNamespace(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, expectedProj, result)
		assert.NotNil(t, result.Namespace)
		assert.Equal(t, "Test Namespace", result.Namespace.Name)
	})

	t.Run("not found", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		deps.mockProjRepo.EXPECT().
			FindByCodeWithNamespace(ctx, "test-ns", "non-existing").
			Return(nil, expectedErr)

		result, err := deps.svc.GetByCodeWithNamespace(ctx, "test-ns", "non-existing")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestProjectService_GetByNamespace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedProjects := []model.Project{
			{ID: 1, ProjectCode: "proj-1", NamespaceCode: "test-ns", Name: "Project 1"},
			{ID: 2, ProjectCode: "proj-2", NamespaceCode: "test-ns", Name: "Project 2"},
		}

		deps.mockProjRepo.EXPECT().
			FindByNamespace(ctx, "test-ns").
			Return(expectedProjects, nil)

		result, err := deps.svc.GetByNamespace(ctx, "test-ns")

		assert.NoError(t, err)
		assert.Equal(t, expectedProjects, result)
	})

	t.Run("empty result", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			FindByNamespace(ctx, "empty-ns").
			Return([]model.Project{}, nil)

		result, err := deps.svc.GetByNamespace(ctx, "empty-ns")

		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		deps.mockProjRepo.EXPECT().
			FindByNamespace(ctx, "test-ns").
			Return(nil, expectedErr)

		result, err := deps.svc.GetByNamespace(ctx, "test-ns")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestProjectService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedProjects := []model.Project{
			{ID: 1, ProjectCode: "proj-1", NamespaceCode: "ns-1", Name: "Project 1"},
			{ID: 2, ProjectCode: "proj-2", NamespaceCode: "ns-2", Name: "Project 2"},
		}

		deps.mockProjRepo.EXPECT().
			FindAll(ctx).
			Return(expectedProjects, nil)

		result, err := deps.svc.GetAll(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expectedProjects, result)
	})

	t.Run("empty result", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			FindAll(ctx).
			Return([]model.Project{}, nil)

		result, err := deps.svc.GetAll(ctx)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		deps.mockProjRepo.EXPECT().
			FindAll(ctx).
			Return(nil, expectedErr)

		result, err := deps.svc.GetAll(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestProjectService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedProjects := []model.Project{
			{ID: 1, ProjectCode: "proj-1", NamespaceCode: "test-ns", Name: "Project 1"},
		}

		deps.mockProjRepo.EXPECT().
			Search(ctx, nil).
			Return(expectedProjects, nil)

		result, err := deps.svc.Search(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, expectedProjects, result)
	})

	t.Run("error", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("search error")

		deps.mockProjRepo.EXPECT().
			Search(ctx, nil).
			Return(nil, expectedErr)

		result, err := deps.svc.Search(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestProjectService_SearchPaginate(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &commonTypes.PaginationInput{
			Limit:  &limit,
			Offset: &offset,
		}
		expectedProjects := []model.Project{
			{ID: 1, ProjectCode: "proj-1", NamespaceCode: "test-ns", Name: "Project 1"},
			{ID: 2, ProjectCode: "proj-2", NamespaceCode: "test-ns", Name: "Project 2"},
		}

		deps.mockProjRepo.EXPECT().
			SearchPaginate(ctx, nil, 10, 5).
			Return(expectedProjects, int64(20), nil)

		result, err := deps.svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 20, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 5, result.Offset)
		assert.Len(t, result.Items, 2)
	})

	t.Run("success with default pagination", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		pagination := &commonTypes.PaginationInput{}
		expectedProjects := []model.Project{
			{ID: 1, ProjectCode: "proj-1", NamespaceCode: "test-ns", Name: "Project 1"},
		}

		deps.mockProjRepo.EXPECT().
			SearchPaginate(ctx, nil, commonTypes.DefaultLimit, commonTypes.DefaultOffset).
			Return(expectedProjects, int64(1), nil)

		result, err := deps.svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, commonTypes.DefaultLimit, result.Limit)
		assert.Equal(t, commonTypes.DefaultOffset, result.Offset)
	})

	t.Run("error", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		pagination := &commonTypes.PaginationInput{}
		expectedErr := errors.New("search error")

		deps.mockProjRepo.EXPECT().
			SearchPaginate(ctx, nil, commonTypes.DefaultLimit, commonTypes.DefaultOffset).
			Return(nil, int64(0), expectedErr)

		result, err := deps.svc.SearchPaginate(ctx, pagination, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestProjectService_CountRedirects(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			CountRedirects(ctx, "test-ns", "test-proj").
			Return(int64(5), nil)

		result, err := deps.svc.CountRedirects(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(5), result)
	})

	t.Run("zero count", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			CountRedirects(ctx, "test-ns", "empty-proj").
			Return(int64(0), nil)

		result, err := deps.svc.CountRedirects(ctx, "test-ns", "empty-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), result)
	})

	t.Run("error", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("count error")

		deps.mockProjRepo.EXPECT().
			CountRedirects(ctx, "test-ns", "test-proj").
			Return(int64(0), expectedErr)

		result, err := deps.svc.CountRedirects(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, int64(0), result)
	})
}

func TestProjectService_CountRedirectDrafts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			CountRedirectDrafts(ctx, "test-ns", "test-proj").
			Return(int64(3), nil)

		result, err := deps.svc.CountRedirectDrafts(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(3), result)
	})

	t.Run("zero count", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			CountRedirectDrafts(ctx, "test-ns", "empty-proj").
			Return(int64(0), nil)

		result, err := deps.svc.CountRedirectDrafts(ctx, "test-ns", "empty-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), result)
	})

	t.Run("error", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("count error")

		deps.mockProjRepo.EXPECT().
			CountRedirectDrafts(ctx, "test-ns", "test-proj").
			Return(int64(0), expectedErr)

		result, err := deps.svc.CountRedirectDrafts(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, int64(0), result)
	})
}

func TestProjectService_CountPages(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			CountPages(ctx, "test-ns", "test-proj").
			Return(int64(5), nil)

		result, err := deps.svc.CountPages(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(5), result)
	})

	t.Run("zero count", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			CountPages(ctx, "test-ns", "empty-proj").
			Return(int64(0), nil)

		result, err := deps.svc.CountPages(ctx, "test-ns", "empty-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), result)
	})

	t.Run("error from repository", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		deps.mockProjRepo.EXPECT().
			CountPages(ctx, "test-ns", "test-proj").
			Return(int64(0), expectedErr)

		result, err := deps.svc.CountPages(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, int64(0), result)
	})
}

func TestProjectService_CountPageDrafts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			CountPageDrafts(ctx, "test-ns", "test-proj").
			Return(int64(3), nil)

		result, err := deps.svc.CountPageDrafts(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(3), result)
	})

	t.Run("zero count", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockProjRepo.EXPECT().
			CountPageDrafts(ctx, "test-ns", "empty-proj").
			Return(int64(0), nil)

		result, err := deps.svc.CountPageDrafts(ctx, "test-ns", "empty-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), result)
	})

	t.Run("error from repository", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		deps.mockProjRepo.EXPECT().
			CountPageDrafts(ctx, "test-ns", "test-proj").
			Return(int64(0), expectedErr)

		result, err := deps.svc.CountPageDrafts(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, int64(0), result)
	})
}

func TestProjectService_TotalPageContentSize(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockPageRepo.EXPECT().
			GetTotalContentSize(ctx, "test-ns", "test-proj").
			Return(int64(1500), nil)

		result, err := deps.svc.TotalPageContentSize(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(1500), result)
	})

	t.Run("zero size", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()

		deps.mockPageRepo.EXPECT().
			GetTotalContentSize(ctx, "test-ns", "empty-proj").
			Return(int64(0), nil)

		result, err := deps.svc.TotalPageContentSize(ctx, "test-ns", "empty-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), result)
	})

	t.Run("error from repository", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		deps.mockPageRepo.EXPECT().
			GetTotalContentSize(ctx, "test-ns", "test-proj").
			Return(int64(0), expectedErr)

		result, err := deps.svc.TotalPageContentSize(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, int64(0), result)
	})
}

func TestProjectService_TotalPageContentSizeLimit(t *testing.T) {
	t.Run("returns configured limit", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		result := deps.svc.TotalPageContentSizeLimit()

		assert.Equal(t, int64(2048), result)
	})
}

func TestProjectService_Publish(t *testing.T) {
	t.Run("project not found", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "non-existing").
			Return(nil, expectedErr)

		result, err := deps.svc.Publish(ctx, "test-ns", "non-existing")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("error counting redirect drafts", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		project := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
			Version:       1,
		}
		expectedErr := errors.New("count error")

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "test-proj").
			Return(project, nil)

		deps.mockProjRepo.EXPECT().
			CountRedirectDrafts(ctx, "test-ns", "test-proj").
			Return(int64(0), expectedErr)

		result, err := deps.svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("error counting page drafts", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		project := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
			Version:       1,
		}
		expectedErr := errors.New("count error")

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "test-proj").
			Return(project, nil)

		deps.mockProjRepo.EXPECT().
			CountRedirectDrafts(ctx, "test-ns", "test-proj").
			Return(int64(1), nil)

		deps.mockProjRepo.EXPECT().
			CountPageDrafts(ctx, "test-ns", "test-proj").
			Return(int64(0), expectedErr)

		result, err := deps.svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("nothing to publish", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		project := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
			Version:       1,
		}

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "test-proj").
			Return(project, nil)

		deps.mockProjRepo.EXPECT().
			CountRedirectDrafts(ctx, "test-ns", "test-proj").
			Return(int64(0), nil)

		deps.mockProjRepo.EXPECT().
			CountPageDrafts(ctx, "test-ns", "test-proj").
			Return(int64(0), nil)

		result, err := deps.svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nothing to publish")
		assert.Nil(t, result)
	})

	t.Run("error getting redirect drafts", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		project := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
			Version:       1,
		}
		expectedErr := errors.New("get drafts error")

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "test-proj").
			Return(project, nil)

		deps.mockProjRepo.EXPECT().
			CountRedirectDrafts(ctx, "test-ns", "test-proj").
			Return(int64(1), nil)

		deps.mockProjRepo.EXPECT().
			CountPageDrafts(ctx, "test-ns", "test-proj").
			Return(int64(0), nil)

		deps.mockRedirectDraft.EXPECT().
			FindByProject(ctx, "test-ns", "test-proj").
			Return(nil, expectedErr)

		result, err := deps.svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("error getting page drafts", func(t *testing.T) {
		deps := setupProjectServiceTest(t)
		defer deps.ctrl.Finish()

		ctx := context.Background()
		project := &model.Project{
			ID:            1,
			ProjectCode:   "test-proj",
			NamespaceCode: "test-ns",
			Name:          "Test Project",
			Version:       1,
		}
		expectedErr := errors.New("get page drafts error")

		deps.mockProjRepo.EXPECT().
			FindByCode(ctx, "test-ns", "test-proj").
			Return(project, nil)

		deps.mockProjRepo.EXPECT().
			CountRedirectDrafts(ctx, "test-ns", "test-proj").
			Return(int64(1), nil)

		deps.mockProjRepo.EXPECT().
			CountPageDrafts(ctx, "test-ns", "test-proj").
			Return(int64(0), nil)

		deps.mockRedirectDraft.EXPECT().
			FindByProject(ctx, "test-ns", "test-proj").
			Return([]model.RedirectDraft{}, nil)

		deps.mockPageDraft.EXPECT().
			FindByProject(ctx, "test-ns", "test-proj").
			Return(nil, expectedErr)

		result, err := deps.svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("success with redirect drafts create/update", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(redirect)
		draft := &model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeCreate, OldRedirectID: &redirect.ID, NewRedirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(draft)

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Version)

		// Check redirect is published
		var publishedRedirect model.Redirect
		db.First(&publishedRedirect, redirect.ID)
		assert.True(t, *publishedRedirect.IsPublished)

		// Check draft is deleted
		var draftCount int64
		db.Model(&model.RedirectDraft{}).Count(&draftCount)
		assert.Equal(t, int64(0), draftCount)
	})

	t.Run("success with redirect drafts delete", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(true), Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(redirect)
		draft := &model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeDelete, OldRedirectID: &redirect.ID}
		db.Create(draft)

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Check redirect is deleted
		var redirectCount int64
		db.Model(&model.Redirect{}).Count(&redirectCount)
		assert.Equal(t, int64(0), redirectCount)
	})

	t.Run("success with page drafts create/update", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), ContentSize: 100, Page: &commonTypes.Page{Path: "/page", Content: "test content"}}
		db.Create(page)
		draft := &model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeCreate, OldPageID: &page.ID, ContentSize: 100, NewPage: &commonTypes.Page{Path: "/page", Content: "test content"}}
		db.Create(draft)

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Check page is published
		var publishedPage model.Page
		db.First(&publishedPage, page.ID)
		assert.True(t, *publishedPage.IsPublished)

		// Check draft is deleted
		var draftCount int64
		db.Model(&model.PageDraft{}).Count(&draftCount)
		assert.Equal(t, int64(0), draftCount)
	})

	t.Run("success with page drafts delete", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(true), ContentSize: 100, Page: &commonTypes.Page{Path: "/page", Content: "test content"}}
		db.Create(page)
		draft := &model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeDelete, OldPageID: &page.ID}
		db.Create(draft)

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Check page is deleted
		var pageCount int64
		db.Model(&model.Page{}).Count(&pageCount)
		assert.Equal(t, int64(0), pageCount)
	})

	t.Run("error saving redirects in transaction", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(redirect)
		draft := &model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeCreate, OldRedirectID: &redirect.ID, NewRedirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(draft)

		errDb := errors.New("forced error")
		// Register callback to fail redirect save
		db.Callback().Create().Before("gorm:create").Register("fail_redirect_save", func(d *gorm.DB) {
			if d.Statement.Table == "redirects" {
				d.Error = errDb
			}
		})

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, err, errDb)
		assert.Nil(t, result)
	})

	t.Run("error delete redirect draft in transaction", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(redirect)
		draft := &model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeCreate, OldRedirectID: &redirect.ID, NewRedirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(draft)
		errDb := errors.New("forced error")
		// Register callback to fail redirect save
		db.Callback().Delete().Before("gorm:delete").Register("fail_redirect_draf_delete", func(d *gorm.DB) {
			if d.Statement.Table == "redirect_drafts" {
				d.Error = errDb
			}
		})

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, err, errDb)
		assert.Nil(t, result)
	})

	t.Run("error delete redirect in transaction", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(redirect)
		draft := &model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeDelete, OldRedirectID: &redirect.ID}
		db.Create(draft)
		errDb := errors.New("forced error")
		// Register callback to fail redirect save
		db.Callback().Delete().Before("gorm:delete").Register("fail_redirect_draf_delete", func(d *gorm.DB) {
			if d.Statement.Table == "redirects" {
				d.Error = errDb
			}
		})

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, err, errDb)
		assert.Nil(t, result)
	})

	t.Run("error saving pages in transaction", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), ContentSize: 100, Page: &commonTypes.Page{Path: "/page", Content: "test content"}}
		db.Create(page)
		draft := &model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeCreate, OldPageID: &page.ID, ContentSize: 100, NewPage: &commonTypes.Page{Path: "/page", Content: "test content"}}
		db.Create(draft)

		errDb := errors.New("forced error")
		// Register callback to fail page save
		db.Callback().Create().Before("gorm:create").Register("fail_page_save", func(d *gorm.DB) {
			if d.Statement.Table == "pages" {
				d.Error = errDb
			}
		})

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, err, errDb)
		assert.Nil(t, result)
	})

	t.Run("error delete page draft in transaction", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), ContentSize: 100, Page: &commonTypes.Page{Path: "/page", Content: "test content"}}
		db.Create(page)
		draft := &model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeCreate, OldPageID: &page.ID, ContentSize: 100, NewPage: &commonTypes.Page{Path: "/page", Content: "test content"}}
		db.Create(draft)

		errDb := errors.New("forced error")
		// Register callback to fail page save
		db.Callback().Delete().Before("gorm:delete").Register("fail_page_save", func(d *gorm.DB) {
			if d.Statement.Table == "page_drafts" {
				d.Error = errDb
			}
		})

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, err, errDb)
		assert.Nil(t, result)
	})

	t.Run("error delete pages in transaction", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), ContentSize: 100, Page: &commonTypes.Page{Path: "/page", Content: "test content"}}
		db.Create(page)
		draft := &model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeDelete, OldPageID: &page.ID, ContentSize: 100}
		db.Create(draft)

		errDb := errors.New("forced error")
		// Register callback to fail page save
		db.Callback().Delete().Before("gorm:delete").Register("fail_page_save", func(d *gorm.DB) {
			if d.Statement.Table == "pages" {
				d.Error = errDb
			}
		})

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, err, errDb)
		assert.Nil(t, result)
	})

	t.Run("error save project in transaction", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), ContentSize: 100, Page: &commonTypes.Page{Path: "/page", Content: "test content"}}
		db.Create(page)
		draft := &model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeDelete, OldPageID: &page.ID, ContentSize: 100}
		db.Create(draft)

		errDb := errors.New("forced error")
		// Register callback to fail page save
		db.Callback().Update().Before("gorm:update").Register("fail_page_save", func(d *gorm.DB) {
			if d.Statement.Table == "projects" {
				d.Error = errDb
			}
		})

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, err, errDb)
		assert.Nil(t, result)
	})
}

func TestProjectService_GetTx(t *testing.T) {
	deps := setupProjectServiceTest(t)
	defer deps.ctrl.Finish()

	ctx := context.Background()
	deps.mockProjRepo.EXPECT().GetTx(ctx).Return(nil)

	result := deps.svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestProjectService_GetQuery(t *testing.T) {
	deps := setupProjectServiceTest(t)
	defer deps.ctrl.Finish()

	ctx := context.Background()
	deps.mockProjRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := deps.svc.GetQuery(ctx)
	assert.Nil(t, result)
}

func TestIsLockError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "SQLite database is locked",
			err:      errors.New("database is locked"),
			expected: true,
		},
		{
			name:     "SQLite database table is locked",
			err:      errors.New("database table is locked"),
			expected: true,
		},
		{
			name:     "PostgreSQL could not obtain lock",
			err:      errors.New("could not obtain lock on row"),
			expected: true,
		},
		{
			name:     "MySQL Lock wait timeout",
			err:      errors.New("Lock wait timeout exceeded"),
			expected: true,
		},
		{
			name:     "MySQL try restarting transaction",
			err:      errors.New("Deadlock found when trying to get lock; try restarting transaction"),
			expected: true,
		},
		{
			name:     "record not found is not a lock error",
			err:      errors.New("record not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLockError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProjectService_Publish_LockError(t *testing.T) {
	t.Run("lock error in transaction returns ErrPublishInProgress", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(redirect)
		draft := &model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeCreate, OldRedirectID: &redirect.ID, NewRedirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(draft)

		// Register callback to simulate lock error
		db.Callback().Query().Before("gorm:query").Register("simulate_lock", func(d *gorm.DB) {
			_, hasForClause := d.Statement.Clauses["FOR"]
			if d.Statement.Table == "projects" && hasForClause {
				d.Error = errors.New("database is locked")
			}
		})

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, ErrPublishInProgress, err)
		assert.Nil(t, result)
	})

	t.Run("non-lock error in lock query is propagated", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Setup data
		ns := &model.Namespace{NamespaceCode: "test-ns", Name: "Test"}
		db.Create(ns)
		proj := &model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 1}
		db.Create(proj)
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: types.Ptr(false), Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(redirect)
		draft := &model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "test-proj", ChangeType: model.DraftChangeTypeCreate, OldRedirectID: &redirect.ID, NewRedirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/old", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent}}
		db.Create(draft)

		expectedErr := errors.New("some other database error")
		// Register callback to simulate a non-lock error
		db.Callback().Query().Before("gorm:query").Register("simulate_other_error", func(d *gorm.DB) {
			_, hasForClause := d.Statement.Clauses["FOR"]
			if d.Statement.Table == "projects" && hasForClause {
				d.Error = expectedErr
			}
		})

		projRepo := repository.NewProjectRepository(db)
		pageRepo := repository.NewPageRepository(db)
		redirectDraftRepo := repository.NewRedirectDraftRepository(db)
		pageDraftRepo := repository.NewPageDraftRepository(db)
		svc := NewProjectService(testContextWithPageConfig(defaultProjectCfg), projRepo, pageRepo, redirectDraftRepo, pageDraftRepo, newTestActivityService(t, db))

		ctx := context.Background()
		result, err := svc.Publish(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestProjectService_PublishActivityEvent(t *testing.T) {
	setup := func(t *testing.T) (*gorm.DB, ProjectService) {
		t.Helper()
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		assert.NoError(t, db.AutoMigrate(
			&model.Namespace{}, &model.Project{},
			&model.Redirect{}, &model.RedirectDraft{},
			&model.Page{}, &model.PageDraft{},
		))

		assert.NoError(t, db.Create(&model.Namespace{NamespaceCode: "test-ns", Name: "Test"}).Error)
		assert.NoError(t, db.Create(&model.Project{ProjectCode: "test-proj", NamespaceCode: "test-ns", Name: "Test", Version: 4}).Error)

		svc := NewProjectService(
			testContextWithPageConfig(defaultProjectCfg),
			repository.NewProjectRepository(db),
			repository.NewPageRepository(db),
			repository.NewRedirectDraftRepository(db),
			repository.NewPageDraftRepository(db),
			newTestActivityService(t, db),
		)
		return db, svc
	}

	// redirectDraft inserts a redirect and a draft of the given change type.
	redirectDraft := func(t *testing.T, db *gorm.DB, changeType model.DraftChangeType, source string) {
		t.Helper()
		redirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   types.Ptr(changeType != model.DraftChangeTypeCreate),
			Redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: source,
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		assert.NoError(t, db.Create(redirect).Error)

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    changeType,
			OldRedirectID: &redirect.ID,
		}
		if changeType != model.DraftChangeTypeDelete {
			draft.NewRedirect = redirect.Redirect
		}
		assert.NoError(t, db.Create(draft).Error)
	}

	pageDraft := func(t *testing.T, db *gorm.DB, changeType model.DraftChangeType, path string) {
		t.Helper()
		page := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   types.Ptr(changeType != model.DraftChangeTypeCreate),
			Page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        path,
				Content:     "body",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		assert.NoError(t, db.Create(page).Error)

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    changeType,
			OldPageID:     &page.ID,
			ContentSize:   4,
		}
		if changeType != model.DraftChangeTypeDelete {
			draft.NewPage = page.Page
		}
		assert.NoError(t, db.Create(draft).Error)
	}

	t.Run("records one event with the new version and the counts", func(t *testing.T) {
		db, svc := setup(t)

		redirectDraft(t, db, model.DraftChangeTypeCreate, "/created")
		redirectDraft(t, db, model.DraftChangeTypeCreate, "/created-2")
		redirectDraft(t, db, model.DraftChangeTypeUpdate, "/updated")
		redirectDraft(t, db, model.DraftChangeTypeDelete, "/deleted")
		pageDraft(t, db, model.DraftChangeTypeUpdate, "/page.txt")

		project, err := svc.Publish(context.Background(), "test-ns", "test-proj")
		assert.NoError(t, err)
		assert.Equal(t, 5, project.Version)

		// Five drafts published, a single journal entry
		assert.Equal(t, int64(1), countActivityEvents(t, db))

		event := lastActivityEvent(t, db)
		assert.Equal(t, model.ActivityResourceProject, event.Resource)
		assert.Equal(t, model.ActivityActionPublish, event.Action)
		assert.Nil(t, event.ResourceID)
		// The version is the one the publish produced, not the one it started from
		assert.JSONEq(t,
			`{"version":5,`+
				`"redirects":{"created":2,"updated":1,"deleted":1},`+
				`"pages":{"created":0,"updated":1,"deleted":0}}`,
			string(event.Data),
		)
	})

	t.Run("a publish with nothing pending records no event", func(t *testing.T) {
		db, svc := setup(t)

		_, err := svc.Publish(context.Background(), "test-ns", "test-proj")
		assert.Error(t, err)
		assert.Zero(t, countActivityEvents(t, db))
	})
}

func TestProjectService_Truncate(t *testing.T) {
	setup := func(t *testing.T) (*gorm.DB, ProjectService) {
		t.Helper()
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		assert.NoError(t, db.AutoMigrate(
			&model.Namespace{}, &model.Project{},
			&model.Redirect{}, &model.RedirectDraft{},
			&model.Page{}, &model.PageDraft{},
		))
		assert.NoError(t, db.Create(&model.Namespace{NamespaceCode: "ns", Name: "NS"}).Error)
		assert.NoError(t, db.Create(&model.Project{ProjectCode: "proj", NamespaceCode: "ns", Name: "P", Version: 7}).Error)

		svc := NewProjectService(
			testContextWithPageConfig(defaultProjectCfg),
			repository.NewProjectRepository(db),
			repository.NewPageRepository(db),
			repository.NewRedirectDraftRepository(db),
			repository.NewPageDraftRepository(db),
			newTestActivityService(t, db),
		)
		return db, svc
	}

	seedRedirect := func(t *testing.T, db *gorm.DB, project string, published bool, withDraft bool) {
		t.Helper()
		redirect := &model.Redirect{
			NamespaceCode: "ns", ProjectCode: project, IsPublished: types.Ptr(published),
			Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/s", Target: "/t", Status: commonTypes.RedirectStatusFound},
		}
		assert.NoError(t, db.Create(redirect).Error)
		if withDraft {
			assert.NoError(t, db.Create(&model.RedirectDraft{
				NamespaceCode: "ns", ProjectCode: project,
				ChangeType: model.DraftChangeTypeUpdate, OldRedirectID: &redirect.ID,
			}).Error)
		}
	}

	countIn := func(t *testing.T, db *gorm.DB, dest any, project string) int64 {
		t.Helper()
		var n int64
		assert.NoError(t, db.Model(dest).Where("namespace_code = ? AND project_code = ?", "ns", project).Count(&n).Error)
		return n
	}

	t.Run("removes published redirects and their drafts", func(t *testing.T) {
		db, svc := setup(t)
		seedRedirect(t, db, "proj", true, true)
		seedRedirect(t, db, "proj", true, false)
		seedRedirect(t, db, "proj", false, true)

		project, err := svc.TruncateRedirects(context.Background(), "ns", "proj")
		assert.NoError(t, err)

		assert.Zero(t, countIn(t, db, &model.Redirect{}, "proj"))
		assert.Zero(t, countIn(t, db, &model.RedirectDraft{}, "proj"))
		assert.Equal(t, 8, project.Version, "the truncate must publish, or agents keep serving what was deleted")
	})

	t.Run("bumps the version so agents re-sync", func(t *testing.T) {
		db, svc := setup(t)
		seedRedirect(t, db, "proj", true, false)

		before := &model.Project{}
		assert.NoError(t, db.Where("project_code = ?", "proj").First(before).Error)

		_, err := svc.TruncateRedirects(context.Background(), "ns", "proj")
		assert.NoError(t, err)

		after := &model.Project{}
		assert.NoError(t, db.Where("project_code = ?", "proj").First(after).Error)
		assert.Equal(t, before.Version+1, after.Version)
		assert.True(t, after.PublishedAt.After(before.PublishedAt))
	})

	t.Run("leaves other projects alone", func(t *testing.T) {
		db, svc := setup(t)
		assert.NoError(t, db.Create(&model.Project{ProjectCode: "other", NamespaceCode: "ns", Name: "O", Version: 1}).Error)
		seedRedirect(t, db, "proj", true, true)
		seedRedirect(t, db, "other", true, true)

		_, err := svc.TruncateRedirects(context.Background(), "ns", "proj")
		assert.NoError(t, err)

		assert.Zero(t, countIn(t, db, &model.Redirect{}, "proj"))
		assert.Equal(t, int64(1), countIn(t, db, &model.Redirect{}, "other"))
		assert.Equal(t, int64(1), countIn(t, db, &model.RedirectDraft{}, "other"))
	})

	t.Run("records one entry with exact counts", func(t *testing.T) {
		db, svc := setup(t)
		seedRedirect(t, db, "proj", true, true)
		seedRedirect(t, db, "proj", true, true)
		seedRedirect(t, db, "proj", true, false)

		_, err := svc.TruncateRedirects(context.Background(), "ns", "proj")
		assert.NoError(t, err)

		event := lastActivityEvent(t, db)
		assert.NotNil(t, event)
		assert.Equal(t, model.ActivityResourceRedirect, event.Resource)
		assert.Equal(t, model.ActivityActionTruncate, event.Action)
		assert.JSONEq(t, `{"published":3,"drafts":2,"version":8}`, string(event.Data))
	})

	t.Run("an empty project is a no-op that still publishes", func(t *testing.T) {
		db, svc := setup(t)

		project, err := svc.TruncateRedirects(context.Background(), "ns", "proj")
		assert.NoError(t, err)
		assert.Equal(t, 8, project.Version)

		event := lastActivityEvent(t, db)
		assert.JSONEq(t, `{"published":0,"drafts":0,"version":8}`, string(event.Data))
	})

	t.Run("pages truncate is independent from redirects", func(t *testing.T) {
		db, svc := setup(t)
		seedRedirect(t, db, "proj", true, true)
		page := &model.Page{
			NamespaceCode: "ns", ProjectCode: "proj", IsPublished: types.Ptr(true),
			Page: &commonTypes.Page{Type: commonTypes.PageTypeBasic, Path: "/p", Content: "x", ContentType: commonTypes.PageContentTypeTextPlain},
		}
		assert.NoError(t, db.Create(page).Error)
		assert.NoError(t, db.Create(&model.PageDraft{
			NamespaceCode: "ns", ProjectCode: "proj",
			ChangeType: model.DraftChangeTypeUpdate, OldPageID: &page.ID,
		}).Error)

		_, err := svc.TruncatePages(context.Background(), "ns", "proj")
		assert.NoError(t, err)

		assert.Zero(t, countIn(t, db, &model.Page{}, "proj"))
		assert.Zero(t, countIn(t, db, &model.PageDraft{}, "proj"))
		assert.Equal(t, int64(1), countIn(t, db, &model.Redirect{}, "proj"), "redirects must be untouched")

		event := lastActivityEvent(t, db)
		assert.Equal(t, model.ActivityResourcePage, event.Resource)
		assert.Equal(t, model.ActivityActionTruncate, event.Action)
	})

	t.Run("nothing is removed when the project does not exist", func(t *testing.T) {
		db, svc := setup(t)
		seedRedirect(t, db, "proj", true, true)

		_, err := svc.TruncateRedirects(context.Background(), "ns", "missing")
		assert.Error(t, err)
		assert.Equal(t, int64(1), countIn(t, db, &model.Redirect{}, "proj"))
	})
}
