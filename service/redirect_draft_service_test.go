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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRedirectDraftServiceTest(t *testing.T) (*gomock.Controller, *mockFlectoRepository.MockRedirectDraftRepository, *gorm.DB, RedirectDraftService) {
	ctrl := gomock.NewController(t)
	mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
	assert.NoError(t, err)
	mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
	svc := NewRedirectDraftService(appContext.TestContext(nil), mockRepo)
	return ctrl, mockRepo, db, svc
}

func TestNewRedirectDraftService(t *testing.T) {
	ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockRepo)
}

func TestRedirectDraftService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedDraft := &model.RedirectDraft{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(expectedDraft, nil)

		result, err := svc.GetByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedDraft, result)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		mockRepo.EXPECT().FindByID(ctx, int64(999)).Return(nil, expectedErr)

		result, err := svc.GetByID(ctx, 999)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestRedirectDraftService_GetByIDWithProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedDraft := &model.RedirectDraft{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"}

		mockRepo.EXPECT().FindByIDWithProject(ctx, "test-ns", "test-proj", int64(1)).Return(expectedDraft, nil)

		result, err := svc.GetByIDWithProject(ctx, "test-ns", "test-proj", 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedDraft, result)
	})
}

func TestRedirectDraftService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		oldRedirectID := int64(10)
		existingDraft := &model.RedirectDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &oldRedirectID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect: &types.Redirect{
				Type:   types.RedirectTypeBasic,
				Source: "/old-source",
				Target: "/old-target",
				Status: types.RedirectStatusMovedPermanent,
			},
		}
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/new-source",
			Target: "/new-target",
			Status: types.RedirectStatusMovedPermanent,
		}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/new-source", &oldRedirectID, gomock.Any()).Return(true, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, draft *model.RedirectDraft) error {
			assert.Equal(t, "/new-source", draft.NewRedirect.Source)
			return nil
		})

		result, err := svc.Update(ctx, 1, newRedirect)

		assert.NoError(t, err)
		assert.Equal(t, "/new-source", result.NewRedirect.Source)
	})

	t.Run("success without source change", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		oldRedirectID := int64(10)
		existingDraft := &model.RedirectDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &oldRedirectID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect: &types.Redirect{
				Type:   types.RedirectTypeBasic,
				Source: "/same-source",
				Target: "/old-target",
				Status: types.RedirectStatusMovedPermanent,
			},
		}
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/same-source",
			Target: "/new-target",
			Status: types.RedirectStatusMovedPermanent,
		}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		// No CheckSourceAvailability call because source didn't change
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		result, err := svc.Update(ctx, 1, newRedirect)

		assert.NoError(t, err)
		assert.Equal(t, "/new-target", result.NewRedirect.Target)
	})

	t.Run("error source already used", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		oldRedirectID := int64(10)
		existingDraft := &model.RedirectDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &oldRedirectID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect: &types.Redirect{
				Source: "/old-source",
			},
		}
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/existing-source",
			Target: "/target",
			Status: types.RedirectStatusMovedPermanent,
		}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/existing-source", &oldRedirectID, gomock.Any()).Return(false, nil)

		result, err := svc.Update(ctx, 1, newRedirect)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrSourceAlreadyUsed)
		assert.Nil(t, result)
	})

	t.Run("error checking source availability on update", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		oldRedirectID := int64(10)
		existingDraft := &model.RedirectDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &oldRedirectID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect: &types.Redirect{
				Source: "/old-source",
			},
		}
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/new-source",
			Target: "/target",
			Status: types.RedirectStatusMovedPermanent,
		}
		expectedErr := errors.New("database error")

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/new-source", &oldRedirectID, gomock.Any()).Return(false, expectedErr)

		result, err := svc.Update(ctx, 1, newRedirect)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		oldRedirectID := int64(10)
		existingDraft := &model.RedirectDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &oldRedirectID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect: &types.Redirect{
				Source: "/old-source",
			},
		}
		newRedirect := &types.Redirect{
			Source: "/new-source",
		}
		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		result, err := svc.Update(ctx, 1, newRedirect)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'Status' failed on the 'required' tag")
		assert.Nil(t, result)
	})

	t.Run("nil newRedirect", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		result, err := svc.Update(ctx, 1, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "newRedirect must be provided")
		assert.Nil(t, result)
	})

	t.Run("draft not found", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newRedirect := &types.Redirect{Source: "/source", Target: "/target"}
		expectedErr := errors.New("record not found")

		mockRepo.EXPECT().FindByID(ctx, int64(999)).Return(nil, expectedErr)

		result, err := svc.Update(ctx, 999, newRedirect)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("cannot update delete draft", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingDraft := &model.RedirectDraft{
			ID:         1,
			ChangeType: model.DraftChangeTypeDelete,
		}
		newRedirect := &types.Redirect{Source: "/source", Target: "/target"}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)

		result, err := svc.Update(ctx, 1, newRedirect)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot update a delete draft")
		assert.Nil(t, result)
	})

	t.Run("update error", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingDraft := &model.RedirectDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/source",
			Target: "/target",
			Status: types.RedirectStatusMovedPermanent,
		}
		expectedErr := errors.New("update failed")

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/source", (*int64)(nil), gomock.Any()).Return(true, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(expectedErr)

		result, err := svc.Update(ctx, 1, newRedirect)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestRedirectDraftService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedDrafts := []model.RedirectDraft{
			{ID: 1, NamespaceCode: "test-ns"},
			{ID: 2, NamespaceCode: "test-ns"},
		}

		mockRepo.EXPECT().Search(ctx, nil).Return(expectedDrafts, nil)

		result, err := svc.Search(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, expectedDrafts, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("search error")

		mockRepo.EXPECT().Search(ctx, nil).Return(nil, expectedErr)

		result, err := svc.Search(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestRedirectDraftService_SearchPaginate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &types.PaginationInput{Limit: &limit, Offset: &offset}
		expectedDrafts := []model.RedirectDraft{
			{ID: 1, NamespaceCode: "test-ns"},
		}

		mockRepo.EXPECT().SearchPaginate(ctx, nil, 10, 5).Return(expectedDrafts, int64(50), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 50, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 5, result.Offset)
		assert.Len(t, result.Items, 1)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedErr := errors.New("search error")

		mockRepo.EXPECT().SearchPaginate(ctx, nil, types.DefaultLimit, types.DefaultOffset).Return(nil, int64(0), expectedErr)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestRedirectDraftService_Create(t *testing.T) {
	t.Run("error when both oldRedirectID and newRedirect are nil", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "oldRedirectID or newRedirect must be provided")
		assert.Nil(t, result)
	})

	t.Run("success create new redirect draft (ChangeType=CREATE)", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/source",
			Target: "/target",
			Status: types.RedirectStatusMovedPermanent,
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/source", (*int64)(nil), (*int64)(nil)).Return(true, nil)
		// Mock FindByID called after creation to reload the draft
		mockRepo.EXPECT().FindByID(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (*model.RedirectDraft, error) {
			var draft model.RedirectDraft
			db.Preload("OldRedirect").First(&draft, id)
			return &draft, nil
		})

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newRedirect)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.DraftChangeTypeCreate, result.ChangeType)
		assert.NotNil(t, result.OldRedirectID)

		// Verify redirect was created
		var redirect model.Redirect
		db.First(&redirect, *result.OldRedirectID)
		assert.Equal(t, "test-ns", redirect.NamespaceCode)
		assert.False(t, *redirect.IsPublished)
	})

	t.Run("success update existing redirect (ChangeType=UPDATE)", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create existing redirect
		isPublished := true
		existingRedirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isPublished,
		}
		db.Create(existingRedirect)

		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/updated-source",
			Target: "/updated-target",
			Status: types.RedirectStatusMovedPermanent,
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/updated-source", &existingRedirect.ID, (*int64)(nil)).Return(true, nil)
		mockRepo.EXPECT().FindByID(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (*model.RedirectDraft, error) {
			var draft model.RedirectDraft
			db.Preload("OldRedirect").First(&draft, id)
			return &draft, nil
		})

		result, err := svc.Create(ctx, "test-ns", "test-proj", &existingRedirect.ID, newRedirect)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.DraftChangeTypeUpdate, result.ChangeType)
		assert.Equal(t, existingRedirect.ID, *result.OldRedirectID)
	})

	t.Run("success delete redirect (ChangeType=DELETE)", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create existing redirect
		isPublished := true
		existingRedirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isPublished,
		}
		db.Create(existingRedirect)

		mockRepo.EXPECT().FindByID(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (*model.RedirectDraft, error) {
			var draft model.RedirectDraft
			db.Preload("OldRedirect").First(&draft, id)
			return &draft, nil
		})

		result, err := svc.Create(ctx, "test-ns", "test-proj", &existingRedirect.ID, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.DraftChangeTypeDelete, result.ChangeType)
	})

	t.Run("error creating redirect in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create a fresh DB with callback to fail redirect creation
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
		assert.NoError(t, err)

		// Register callback to fail redirect creation
		db.Callback().Create().Before("gorm:create").Register("fail_redirect", func(d *gorm.DB) {
			if d.Statement.Table == "redirects" {
				d.Error = errors.New("forced redirect creation error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewRedirectDraftService(appContext.TestContext(nil), mockRepo)

		ctx := context.Background()
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/source",
			Target: "/target",
			Status: types.RedirectStatusMovedPermanent,
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/source", (*int64)(nil), (*int64)(nil)).Return(true, nil)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newRedirect)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced redirect creation error")
		assert.Nil(t, result)
	})

	t.Run("error creating draft in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create a fresh DB with callback to fail draft creation
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
		assert.NoError(t, err)

		// Register callback to fail only redirect_draft creation
		db.Callback().Create().Before("gorm:create").Register("fail_draft", func(d *gorm.DB) {
			if d.Statement.Table == "redirect_drafts" {
				d.Error = errors.New("forced draft creation error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewRedirectDraftService(appContext.TestContext(nil), mockRepo)

		ctx := context.Background()
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/source",
			Target: "/target",
			Status: types.RedirectStatusMovedPermanent,
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/source", (*int64)(nil), (*int64)(nil)).Return(true, nil)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newRedirect)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced draft creation error")
		assert.Nil(t, result)
	})

	t.Run("error source already used on create", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/existing-source",
			Target: "/target",
			Status: types.RedirectStatusMovedPermanent,
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/existing-source", (*int64)(nil), (*int64)(nil)).Return(false, nil)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newRedirect)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrSourceAlreadyUsed)
		assert.Nil(t, result)
	})

	t.Run("error checking source availability on create", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/source",
			Target: "/target",
			Status: types.RedirectStatusMovedPermanent,
		}
		expectedErr := errors.New("database error")

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/source", (*int64)(nil), (*int64)(nil)).Return(false, expectedErr)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newRedirect)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newRedirect := &types.Redirect{
			Type:   types.RedirectTypeBasic,
			Source: "/source",
			Status: types.RedirectStatusMovedPermanent,
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "test-ns", "test-proj", "/source", (*int64)(nil), (*int64)(nil)).Return(true, nil)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newRedirect)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'Target' failed on the 'required' tag")
		assert.Nil(t, result)
	})
}

func TestRedirectDraftService_Delete(t *testing.T) {
	t.Run("error when draft not found", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		mockRepo.EXPECT().FindByID(ctx, int64(999)).Return(nil, expectedErr)

		result, err := svc.Delete(ctx, 999)

		assert.Error(t, err)
		assert.False(t, result)
	})

	t.Run("success delete UPDATE draft (keeps redirect)", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create redirect and draft
		isPublished := true
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify draft is deleted
		var foundDraft model.RedirectDraft
		err = db.First(&foundDraft, draft.ID).Error
		assert.Error(t, err)

		// Verify redirect still exists
		var foundRedirect model.Redirect
		err = db.First(&foundRedirect, redirect.ID).Error
		assert.NoError(t, err)
	})

	t.Run("success delete CREATE draft (deletes redirect too)", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create redirect and draft with ChangeType=CREATE
		isPublished := false
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(draft)

		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify draft is deleted
		var foundDraft model.RedirectDraft
		err = db.First(&foundDraft, draft.ID).Error
		assert.Error(t, err)

		// Verify redirect is also deleted
		var foundRedirect model.Redirect
		err = db.First(&foundRedirect, redirect.ID).Error
		assert.Error(t, err)
	})

	t.Run("success delete DELETE draft (keeps redirect)", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create redirect and draft with ChangeType=DELETE
		isPublished := true
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeDelete,
		}
		db.Create(draft)

		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify draft is deleted
		var foundDraft model.RedirectDraft
		err = db.First(&foundDraft, draft.ID).Error
		assert.Error(t, err)

		// Verify redirect still exists
		var foundRedirect model.Redirect
		err = db.First(&foundRedirect, redirect.ID).Error
		assert.NoError(t, err)
	})

	t.Run("error deleting draft in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create a fresh DB with callback to fail draft deletion
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
		assert.NoError(t, err)

		// Create redirect and draft
		isPublished := true
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		// Register callback to fail draft deletion
		db.Callback().Delete().Before("gorm:delete").Register("fail_draft_delete", func(d *gorm.DB) {
			if d.Statement.Table == "redirect_drafts" {
				d.Error = errors.New("forced draft deletion error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewRedirectDraftService(appContext.TestContext(nil), mockRepo)

		ctx := context.Background()
		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced draft deletion error")
		assert.False(t, result)
	})

	t.Run("error deleting redirect in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Create a fresh DB with callback to fail redirect deletion
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
		assert.NoError(t, err)

		// Create redirect and draft with ChangeType=CREATE
		isPublished := false
		redirect := &model.Redirect{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(draft)

		// Register callback to fail redirect deletion only
		db.Callback().Delete().Before("gorm:delete").Register("fail_redirect_delete", func(d *gorm.DB) {
			if d.Statement.Table == "redirects" {
				d.Error = errors.New("forced redirect deletion error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewRedirectDraftService(appContext.TestContext(nil), mockRepo)

		ctx := context.Background()
		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced redirect deletion error")
		assert.False(t, result)
	})
}

func TestRedirectDraftService_Rollback(t *testing.T) {
	t.Run("success deletes drafts and unpublished redirects", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create published redirect (should be kept)
		isPublished := true
		publishedRedirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isPublished,
		}
		db.Create(publishedRedirect)

		// Create unpublished redirect (should be deleted)
		isUnpublished := false
		unpublishedRedirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isUnpublished,
		}
		db.Create(unpublishedRedirect)

		// Create drafts (should be deleted)
		draft1 := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &publishedRedirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft1)

		draft2 := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &unpublishedRedirect.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(draft2)

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify drafts are deleted
		var draftCount int64
		db.Model(&model.RedirectDraft{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "test-proj").Count(&draftCount)
		assert.Equal(t, int64(0), draftCount)

		// Verify unpublished redirect is deleted
		var unpublishedCount int64
		db.Model(&model.Redirect{}).Where("id = ?", unpublishedRedirect.ID).Count(&unpublishedCount)
		assert.Equal(t, int64(0), unpublishedCount)

		// Verify published redirect is kept
		var publishedCount int64
		db.Model(&model.Redirect{}).Where("id = ?", publishedRedirect.ID).Count(&publishedCount)
		assert.Equal(t, int64(1), publishedCount)
	})

	t.Run("success with no drafts or unpublished redirects", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create only a published redirect
		isPublished := true
		redirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isPublished,
		}
		db.Create(redirect)

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify published redirect is kept
		var count int64
		db.Model(&model.Redirect{}).Where("id = ?", redirect.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("success only affects specified project", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create draft in different project (should be kept)
		isUnpublished := false
		otherProjectRedirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "other-proj",
			IsPublished:   &isUnpublished,
		}
		db.Create(otherProjectRedirect)

		otherProjectDraft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "other-proj",
			OldRedirectID: &otherProjectRedirect.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(otherProjectDraft)

		// Create draft in target project (should be deleted)
		targetRedirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isUnpublished,
		}
		db.Create(targetRedirect)

		targetDraft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &targetRedirect.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(targetDraft)

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify target project draft is deleted
		var targetDraftCount int64
		db.Model(&model.RedirectDraft{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "test-proj").Count(&targetDraftCount)
		assert.Equal(t, int64(0), targetDraftCount)

		// Verify other project draft is kept
		var otherDraftCount int64
		db.Model(&model.RedirectDraft{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "other-proj").Count(&otherDraftCount)
		assert.Equal(t, int64(1), otherDraftCount)

		// Verify other project redirect is kept
		var otherRedirectCount int64
		db.Model(&model.Redirect{}).Where("id = ?", otherProjectRedirect.ID).Count(&otherRedirectCount)
		assert.Equal(t, int64(1), otherRedirectCount)
	})

	t.Run("error deleting drafts in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
		assert.NoError(t, err)

		// Register callback to fail draft deletion
		db.Callback().Delete().Before("gorm:delete").Register("fail_rollback_draft", func(d *gorm.DB) {
			if d.Statement.Table == "redirect_drafts" {
				d.Error = errors.New("forced draft deletion error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewRedirectDraftService(appContext.TestContext(nil), mockRepo)

		ctx := context.Background()

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced draft deletion error")
		assert.False(t, result)
	})

	t.Run("error deleting redirects in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
		assert.NoError(t, err)

		// Register callback to fail redirect deletion only
		db.Callback().Delete().Before("gorm:delete").Register("fail_rollback_redirect", func(d *gorm.DB) {
			if d.Statement.Table == "redirects" {
				d.Error = errors.New("forced redirect deletion error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewRedirectDraftService(appContext.TestContext(nil), mockRepo)

		ctx := context.Background()

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced redirect deletion error")
		assert.False(t, result)
	})
}

func TestRedirectDraftService_GetTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
	svc := NewRedirectDraftService(appContext.TestContext(nil), mockRepo)

	ctx := context.Background()
	mockRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestRedirectDraftService_GetQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
	svc := NewRedirectDraftService(appContext.TestContext(nil), mockRepo)

	ctx := context.Background()
	mockRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}
