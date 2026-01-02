package service

import (
	"context"
	"errors"
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/config"
	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var defaultPageDraftTestConfig = config.PageConfig{
	SizeLimit:      1024,       // 1KB
	TotalSizeLimit: 1024 * 100, // 100KB
}

func testContextWithPageConfig(pageConfig config.PageConfig) *appContext.Context {
	ctx := appContext.TestContext(nil)
	ctx.Config.Page = pageConfig
	return ctx
}

func setupPageDraftServiceTest(t *testing.T) (*gomock.Controller, *mockFlectoRepository.MockPageDraftRepository, *mockFlectoRepository.MockPageRepository, *gorm.DB, PageDraftService) {
	ctrl := gomock.NewController(t)
	mockRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
	mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Page{}, &model.PageDraft{})
	assert.NoError(t, err)
	mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
	svc := NewPageDraftService(testContextWithPageConfig(defaultPageDraftTestConfig), mockRepo, mockPageRepo)
	return ctrl, mockRepo, mockPageRepo, db, svc
}

func TestNewPageDraftService(t *testing.T) {
	ctrl, mockRepo, mockPageRepo, _, svc := setupPageDraftServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockRepo)
	assert.NotNil(t, mockPageRepo)
}

func TestPageDraftService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedDraft := &model.PageDraft{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(expectedDraft, nil)

		result, err := svc.GetByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedDraft, result)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		mockRepo.EXPECT().FindByID(ctx, int64(999)).Return(nil, expectedErr)

		result, err := svc.GetByID(ctx, 999)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPageDraftService_GetByIDWithProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedDraft := &model.PageDraft{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"}

		mockRepo.EXPECT().FindByIDWithProject(ctx, "test-ns", "test-proj", int64(1)).Return(expectedDraft, nil)

		result, err := svc.GetByIDWithProject(ctx, "test-ns", "test-proj", 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedDraft, result)
	})
}

func TestPageDraftService_Create(t *testing.T) {
	t.Run("error when both oldPageID and newPage are nil", func(t *testing.T) {
		ctrl, _, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "oldPageID or newPage must be provided")
		assert.Nil(t, result)
	})

	t.Run("success create new page draft (ChangeType=CREATE)", func(t *testing.T) {
		ctrl, mockRepo, mockPageRepo, db, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/robots.txt",
			Content:     "User-agent: *\nDisallow:",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/robots.txt", (*int64)(nil), (*int64)(nil)).Return(true, nil)
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(0), nil)
		mockRepo.EXPECT().FindByID(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (*model.PageDraft, error) {
			var draft model.PageDraft
			db.Preload("OldPage").First(&draft, id)
			return &draft, nil
		})

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newPage)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.DraftChangeTypeCreate, result.ChangeType)
		assert.NotNil(t, result.OldPageID)

		// Verify page was created
		var page model.Page
		db.First(&page, *result.OldPageID)
		assert.Equal(t, "test-ns", page.NamespaceCode)
		assert.False(t, *page.IsPublished)
	})

	t.Run("success update existing page (ChangeType=UPDATE)", func(t *testing.T) {
		ctrl, mockRepo, mockPageRepo, db, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create existing page
		isPublished := true
		existingPage := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isPublished,
		}
		db.Create(existingPage)

		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/updated-robots.txt",
			Content:     "User-agent: *\nDisallow: /admin",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/updated-robots.txt", &existingPage.ID, (*int64)(nil)).Return(true, nil)
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(0), nil)
		mockRepo.EXPECT().FindByID(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (*model.PageDraft, error) {
			var draft model.PageDraft
			db.Preload("OldPage").First(&draft, id)
			return &draft, nil
		})

		result, err := svc.Create(ctx, "test-ns", "test-proj", &existingPage.ID, newPage)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.DraftChangeTypeUpdate, result.ChangeType)
		assert.Equal(t, existingPage.ID, *result.OldPageID)
	})

	t.Run("success delete page (ChangeType=DELETE)", func(t *testing.T) {
		ctrl, mockRepo, _, db, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create existing page
		isPublished := true
		existingPage := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isPublished,
		}
		db.Create(existingPage)

		mockRepo.EXPECT().FindByID(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (*model.PageDraft, error) {
			var draft model.PageDraft
			db.Preload("OldPage").First(&draft, id)
			return &draft, nil
		})

		result, err := svc.Create(ctx, "test-ns", "test-proj", &existingPage.ID, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.DraftChangeTypeDelete, result.ChangeType)
	})

	t.Run("error content size exceeded", func(t *testing.T) {
		ctrl, _, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		// Create content larger than 1KB limit
		largeContent := make([]byte, 2048)
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/large.txt",
			Content:     string(largeContent),
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newPage)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContentSizeExceeded)
		assert.Nil(t, result)
	})

	t.Run("error path already used", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/existing-path.txt",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/existing-path.txt", (*int64)(nil), (*int64)(nil)).Return(false, nil)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newPage)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPathAlreadyUsed)
		assert.Nil(t, result)
	})

	t.Run("error checking path availability", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/test.txt",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}
		expectedErr := errors.New("database error")

		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/test.txt", (*int64)(nil), (*int64)(nil)).Return(false, expectedErr)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newPage)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("error total size limit reached", func(t *testing.T) {
		ctrl, mockRepo, mockPageRepo, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/test.txt",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/test.txt", (*int64)(nil), (*int64)(nil)).Return(true, nil)
		// Return a size that when added to new content exceeds the 100KB limit
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(1024*100), nil)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newPage)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTotalSizeLimitReached)
		assert.Nil(t, result)
	})

	t.Run("error getting total content size", func(t *testing.T) {
		ctrl, mockRepo, mockPageRepo, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/test.txt",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}
		expectedErr := errors.New("database error")

		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/test.txt", (*int64)(nil), (*int64)(nil)).Return(true, nil)
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(0), expectedErr)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newPage)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		ctrl, mockRepo, mockPageRepo, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newPage := &commonTypes.Page{
			Type: commonTypes.PageTypeBasic,
			// Missing required fields
		}

		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "", (*int64)(nil), (*int64)(nil)).Return(true, nil)
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(0), nil)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newPage)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation")
		assert.Nil(t, result)
	})

	t.Run("error creating page in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Register callback to fail page creation
		db.Callback().Create().Before("gorm:create").Register("fail_page", func(d *gorm.DB) {
			if d.Statement.Table == "pages" {
				d.Error = errors.New("forced page creation error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
		mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewPageDraftService(testContextWithPageConfig(defaultPageDraftTestConfig), mockRepo, mockPageRepo)

		ctx := context.Background()
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/test.txt",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/test.txt", (*int64)(nil), (*int64)(nil)).Return(true, nil)
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(0), nil)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newPage)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced page creation error")
		assert.Nil(t, result)
	})

	t.Run("error creating draft in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Register callback to fail only page_draft creation
		db.Callback().Create().Before("gorm:create").Register("fail_draft", func(d *gorm.DB) {
			if d.Statement.Table == "page_drafts" {
				d.Error = errors.New("forced draft creation error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
		mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewPageDraftService(testContextWithPageConfig(defaultPageDraftTestConfig), mockRepo, mockPageRepo)

		ctx := context.Background()
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/test.txt",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/test.txt", (*int64)(nil), (*int64)(nil)).Return(true, nil)
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(0), nil)

		result, err := svc.Create(ctx, "test-ns", "test-proj", nil, newPage)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced draft creation error")
		assert.Nil(t, result)
	})
}

func TestPageDraftService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		oldPageID := int64(10)
		existingDraft := &model.PageDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &oldPageID,
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   100,
			NewPage: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        "/old-path.txt",
				Content:     "old content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/new-path.txt",
			Content:     "new content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/new-path.txt", &oldPageID, gomock.Any()).Return(true, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, draft *model.PageDraft) error {
			assert.Equal(t, "/new-path.txt", draft.NewPage.Path)
			return nil
		})

		result, err := svc.Update(ctx, 1, newPage)

		assert.NoError(t, err)
		assert.Equal(t, "/new-path.txt", result.NewPage.Path)
	})

	t.Run("success without path change", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		oldPageID := int64(10)
		existingDraft := &model.PageDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &oldPageID,
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   100,
			NewPage: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        "/same-path.txt",
				Content:     "old content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/same-path.txt",
			Content:     "new content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		// No CheckPathAvailability call because path didn't change
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		result, err := svc.Update(ctx, 1, newPage)

		assert.NoError(t, err)
		assert.Equal(t, "new content", result.NewPage.Content)
	})

	t.Run("error path already used", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		oldPageID := int64(10)
		existingDraft := &model.PageDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &oldPageID,
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   100,
			NewPage: &commonTypes.Page{
				Path: "/old-path.txt",
			},
		}
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/existing-path.txt",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/existing-path.txt", &oldPageID, gomock.Any()).Return(false, nil)

		result, err := svc.Update(ctx, 1, newPage)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPathAlreadyUsed)
		assert.Nil(t, result)
	})

	t.Run("error checking path availability", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		oldPageID := int64(10)
		existingDraft := &model.PageDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &oldPageID,
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   100,
			NewPage: &commonTypes.Page{
				Path: "/old-path.txt",
			},
		}
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/new-path.txt",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}
		expectedErr := errors.New("database error")

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		mockRepo.EXPECT().CheckPathAvailability(ctx, "test-ns", "test-proj", "/new-path.txt", &oldPageID, gomock.Any()).Return(false, expectedErr)

		result, err := svc.Update(ctx, 1, newPage)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("error content size exceeded", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingDraft := &model.PageDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   100,
			NewPage: &commonTypes.Page{
				Path: "/path.txt",
			},
		}
		// Create content larger than 1KB limit
		largeContent := make([]byte, 2048)
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/path.txt",
			Content:     string(largeContent),
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)

		result, err := svc.Update(ctx, 1, newPage)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContentSizeExceeded)
		assert.Nil(t, result)
	})

	t.Run("error total size limit reached on content increase", func(t *testing.T) {
		ctrl, mockRepo, mockPageRepo, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingDraft := &model.PageDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   10, // Small existing size
			NewPage: &commonTypes.Page{
				Path: "/path.txt",
			},
		}
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/path.txt",
			Content:     "larger content that increases size",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		// Current total is close to limit, the difference would exceed it
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(1024*100-10), nil)

		result, err := svc.Update(ctx, 1, newPage)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTotalSizeLimitReached)
		assert.Nil(t, result)
	})

	t.Run("nil newPage", func(t *testing.T) {
		ctrl, _, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		result, err := svc.Update(ctx, 1, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "newPage must be provided")
		assert.Nil(t, result)
	})

	t.Run("draft not found", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		newPage := &commonTypes.Page{Path: "/path.txt", Content: "content"}
		expectedErr := errors.New("record not found")

		mockRepo.EXPECT().FindByID(ctx, int64(999)).Return(nil, expectedErr)

		result, err := svc.Update(ctx, 999, newPage)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("cannot update delete draft", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingDraft := &model.PageDraft{
			ID:         1,
			ChangeType: model.DraftChangeTypeDelete,
		}
		newPage := &commonTypes.Page{Path: "/path.txt", Content: "content"}

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)

		result, err := svc.Update(ctx, 1, newPage)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot update a delete draft")
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingDraft := &model.PageDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   100,
			NewPage: &commonTypes.Page{
				Path: "/old-path.txt",
			},
		}
		newPage := &commonTypes.Page{
			Path: "/new-path.txt",
			// Missing required fields
		}
		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)

		result, err := svc.Update(ctx, 1, newPage)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation")
		assert.Nil(t, result)
	})

	t.Run("update error", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingDraft := &model.PageDraft{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   100,
			NewPage: &commonTypes.Page{
				Path: "/path.txt",
			},
		}
		newPage := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/path.txt",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}
		expectedErr := errors.New("update failed")

		mockRepo.EXPECT().FindByID(ctx, int64(1)).Return(existingDraft, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(expectedErr)

		result, err := svc.Update(ctx, 1, newPage)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestPageDraftService_Delete(t *testing.T) {
	t.Run("error when draft not found", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		mockRepo.EXPECT().FindByID(ctx, int64(999)).Return(nil, expectedErr)

		result, err := svc.Delete(ctx, 999)

		assert.Error(t, err)
		assert.False(t, result)
	})

	t.Run("success delete UPDATE draft (keeps page)", func(t *testing.T) {
		ctrl, mockRepo, _, db, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create page and draft
		isPublished := true
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(page)

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify draft is deleted
		var foundDraft model.PageDraft
		err = db.First(&foundDraft, draft.ID).Error
		assert.Error(t, err)

		// Verify page still exists
		var foundPage model.Page
		err = db.First(&foundPage, page.ID).Error
		assert.NoError(t, err)
	})

	t.Run("success delete CREATE draft (deletes page too)", func(t *testing.T) {
		ctrl, mockRepo, _, db, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create page and draft with ChangeType=CREATE
		isPublished := false
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(page)

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(draft)

		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify draft is deleted
		var foundDraft model.PageDraft
		err = db.First(&foundDraft, draft.ID).Error
		assert.Error(t, err)

		// Verify page is also deleted
		var foundPage model.Page
		err = db.First(&foundPage, page.ID).Error
		assert.Error(t, err)
	})

	t.Run("success delete DELETE draft (keeps page)", func(t *testing.T) {
		ctrl, mockRepo, _, db, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create page and draft with ChangeType=DELETE
		isPublished := true
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(page)

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeDelete,
		}
		db.Create(draft)

		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify draft is deleted
		var foundDraft model.PageDraft
		err = db.First(&foundDraft, draft.ID).Error
		assert.Error(t, err)

		// Verify page still exists
		var foundPage model.Page
		err = db.First(&foundPage, page.ID).Error
		assert.NoError(t, err)
	})

	t.Run("error deleting draft in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Create page and draft
		isPublished := true
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(page)

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		// Register callback to fail draft deletion
		db.Callback().Delete().Before("gorm:delete").Register("fail_draft_delete", func(d *gorm.DB) {
			if d.Statement.Table == "page_drafts" {
				d.Error = errors.New("forced draft deletion error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
		mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewPageDraftService(testContextWithPageConfig(defaultPageDraftTestConfig), mockRepo, mockPageRepo)

		ctx := context.Background()
		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced draft deletion error")
		assert.False(t, result)
	})

	t.Run("error deleting page in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Create page and draft with ChangeType=CREATE
		isPublished := false
		page := &model.Page{NamespaceCode: "test-ns", ProjectCode: "test-proj", IsPublished: &isPublished}
		db.Create(page)

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(draft)

		// Register callback to fail page deletion only
		db.Callback().Delete().Before("gorm:delete").Register("fail_page_delete", func(d *gorm.DB) {
			if d.Statement.Table == "pages" {
				d.Error = errors.New("forced page deletion error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
		mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewPageDraftService(testContextWithPageConfig(defaultPageDraftTestConfig), mockRepo, mockPageRepo)

		ctx := context.Background()
		mockRepo.EXPECT().FindByID(ctx, draft.ID).Return(draft, nil)

		result, err := svc.Delete(ctx, draft.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced page deletion error")
		assert.False(t, result)
	})
}

func TestPageDraftService_Rollback(t *testing.T) {
	t.Run("success deletes drafts and unpublished pages", func(t *testing.T) {
		ctrl, _, _, db, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create published page (should be kept)
		isPublished := true
		publishedPage := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isPublished,
		}
		db.Create(publishedPage)

		// Create unpublished page (should be deleted)
		isUnpublished := false
		unpublishedPage := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isUnpublished,
		}
		db.Create(unpublishedPage)

		// Create drafts (should be deleted)
		draft1 := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &publishedPage.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft1)

		draft2 := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &unpublishedPage.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(draft2)

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify drafts are deleted
		var draftCount int64
		db.Model(&model.PageDraft{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "test-proj").Count(&draftCount)
		assert.Equal(t, int64(0), draftCount)

		// Verify unpublished page is deleted
		var unpublishedCount int64
		db.Model(&model.Page{}).Where("id = ?", unpublishedPage.ID).Count(&unpublishedCount)
		assert.Equal(t, int64(0), unpublishedCount)

		// Verify published page is kept
		var publishedCount int64
		db.Model(&model.Page{}).Where("id = ?", publishedPage.ID).Count(&publishedCount)
		assert.Equal(t, int64(1), publishedCount)
	})

	t.Run("success with no drafts or unpublished pages", func(t *testing.T) {
		ctrl, _, _, db, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create only a published page
		isPublished := true
		page := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isPublished,
		}
		db.Create(page)

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify published page is kept
		var count int64
		db.Model(&model.Page{}).Where("id = ?", page.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("success only affects specified project", func(t *testing.T) {
		ctrl, _, _, db, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create draft in different project (should be kept)
		isUnpublished := false
		otherProjectPage := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "other-proj",
			IsPublished:   &isUnpublished,
		}
		db.Create(otherProjectPage)

		otherProjectDraft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "other-proj",
			OldPageID:     &otherProjectPage.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(otherProjectDraft)

		// Create draft in target project (should be deleted)
		targetPage := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   &isUnpublished,
		}
		db.Create(targetPage)

		targetDraft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &targetPage.ID,
			ChangeType:    model.DraftChangeTypeCreate,
		}
		db.Create(targetDraft)

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.True(t, result)

		// Verify target project draft is deleted
		var targetDraftCount int64
		db.Model(&model.PageDraft{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "test-proj").Count(&targetDraftCount)
		assert.Equal(t, int64(0), targetDraftCount)

		// Verify other project draft is kept
		var otherDraftCount int64
		db.Model(&model.PageDraft{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "other-proj").Count(&otherDraftCount)
		assert.Equal(t, int64(1), otherDraftCount)

		// Verify other project page is kept
		var otherPageCount int64
		db.Model(&model.Page{}).Where("id = ?", otherProjectPage.ID).Count(&otherPageCount)
		assert.Equal(t, int64(1), otherPageCount)
	})

	t.Run("error deleting drafts in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Register callback to fail draft deletion
		db.Callback().Delete().Before("gorm:delete").Register("fail_rollback_draft", func(d *gorm.DB) {
			if d.Statement.Table == "page_drafts" {
				d.Error = errors.New("forced draft deletion error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
		mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewPageDraftService(testContextWithPageConfig(defaultPageDraftTestConfig), mockRepo, mockPageRepo)

		ctx := context.Background()

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced draft deletion error")
		assert.False(t, result)
	})

	t.Run("error deleting pages in transaction", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Page{}, &model.PageDraft{})
		assert.NoError(t, err)

		// Register callback to fail page deletion only
		db.Callback().Delete().Before("gorm:delete").Register("fail_rollback_page", func(d *gorm.DB) {
			if d.Statement.Table == "pages" {
				d.Error = errors.New("forced page deletion error")
			}
		})

		mockRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
		mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
		mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
		svc := NewPageDraftService(testContextWithPageConfig(defaultPageDraftTestConfig), mockRepo, mockPageRepo)

		ctx := context.Background()

		result, err := svc.Rollback(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced page deletion error")
		assert.False(t, result)
	})
}

func TestPageDraftService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedDrafts := []model.PageDraft{
			{ID: 1, NamespaceCode: "test-ns"},
			{ID: 2, NamespaceCode: "test-ns"},
		}

		mockRepo.EXPECT().Search(ctx, nil).Return(expectedDrafts, nil)

		result, err := svc.Search(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, expectedDrafts, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("search error")

		mockRepo.EXPECT().Search(ctx, nil).Return(nil, expectedErr)

		result, err := svc.Search(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPageDraftService_SearchPaginate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &commonTypes.PaginationInput{Limit: &limit, Offset: &offset}
		expectedDrafts := []model.PageDraft{
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
		ctrl, mockRepo, _, _, svc := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &commonTypes.PaginationInput{}
		expectedErr := errors.New("search error")

		mockRepo.EXPECT().SearchPaginate(ctx, nil, commonTypes.DefaultLimit, commonTypes.DefaultOffset).Return(nil, int64(0), expectedErr)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPageDraftService_checkTotalSizeLimit(t *testing.T) {
	t.Run("success within limit", func(t *testing.T) {
		ctrl, _, mockPageRepo, _, _ := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		svc := &pageDraftService{
			ctx:      testContextWithPageConfig(defaultPageDraftTestConfig),
			pageRepo: mockPageRepo,
		}

		ctx := context.Background()
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(1024*50), nil)

		err := svc.checkTotalSizeLimit(ctx, "test-ns", "test-proj", 1024)

		assert.NoError(t, err)
	})

	t.Run("error exceeds limit", func(t *testing.T) {
		ctrl, _, mockPageRepo, _, _ := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		svc := &pageDraftService{
			ctx:      testContextWithPageConfig(defaultPageDraftTestConfig),
			pageRepo: mockPageRepo,
		}

		ctx := context.Background()
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(1024*100), nil)

		err := svc.checkTotalSizeLimit(ctx, "test-ns", "test-proj", 1)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTotalSizeLimitReached)
	})

	t.Run("error getting total size", func(t *testing.T) {
		ctrl, _, mockPageRepo, _, _ := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		svc := &pageDraftService{
			ctx:      testContextWithPageConfig(defaultPageDraftTestConfig),
			pageRepo: mockPageRepo,
		}

		ctx := context.Background()
		expectedErr := errors.New("database error")
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(0), expectedErr)

		err := svc.checkTotalSizeLimit(ctx, "test-ns", "test-proj", 1024)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestPageDraftService_checkTotalSizeLimitDiff(t *testing.T) {
	t.Run("success within limit", func(t *testing.T) {
		ctrl, _, mockPageRepo, _, _ := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		svc := &pageDraftService{
			ctx:      testContextWithPageConfig(defaultPageDraftTestConfig),
			pageRepo: mockPageRepo,
		}

		ctx := context.Background()
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(1024*50), nil)

		err := svc.checkTotalSizeLimitDiff(ctx, "test-ns", "test-proj", 100)

		assert.NoError(t, err)
	})

	t.Run("error exceeds limit", func(t *testing.T) {
		ctrl, _, mockPageRepo, _, _ := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		svc := &pageDraftService{
			ctx:      testContextWithPageConfig(defaultPageDraftTestConfig),
			pageRepo: mockPageRepo,
		}

		ctx := context.Background()
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(1024*100-10), nil)

		err := svc.checkTotalSizeLimitDiff(ctx, "test-ns", "test-proj", 20)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTotalSizeLimitReached)
	})

	t.Run("error getting total content size", func(t *testing.T) {
		ctrl, _, mockPageRepo, _, _ := setupPageDraftServiceTest(t)
		defer ctrl.Finish()

		svc := &pageDraftService{
			ctx:      testContextWithPageConfig(defaultPageDraftTestConfig),
			pageRepo: mockPageRepo,
		}

		ctx := context.Background()
		expectedErr := errors.New("database error")
		mockPageRepo.EXPECT().GetTotalContentSize(ctx, "test-ns", "test-proj").Return(int64(0), expectedErr)

		err := svc.checkTotalSizeLimitDiff(ctx, "test-ns", "test-proj", 100)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestPageDraftService_GetTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
	mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
	svc := NewPageDraftService(testContextWithPageConfig(defaultPageDraftTestConfig), mockRepo, mockPageRepo)

	ctx := context.Background()
	mockRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestPageDraftService_GetQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockFlectoRepository.NewMockPageDraftRepository(ctrl)
	mockPageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)
	svc := NewPageDraftService(testContextWithPageConfig(defaultPageDraftTestConfig), mockRepo, mockPageRepo)

	ctx := context.Background()
	mockRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}
