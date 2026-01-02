package repository

import (
	"context"
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPageDraftTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Page{}, &model.PageDraft{})
	assert.NoError(t, err)

	return db
}

func createTestPageDraftNamespace(t *testing.T, db *gorm.DB, code, name string) *model.Namespace {
	ns := &model.Namespace{NamespaceCode: code, Name: name}
	err := db.Create(ns).Error
	assert.NoError(t, err)
	return ns
}

func createTestPageDraftProject(t *testing.T, db *gorm.DB, namespaceCode, projectCode, name string) *model.Project {
	proj := &model.Project{NamespaceCode: namespaceCode, ProjectCode: projectCode, Name: name}
	err := db.Create(proj).Error
	assert.NoError(t, err)
	return proj
}

func createTestPage(t *testing.T, db *gorm.DB, namespaceCode, projectCode string) *model.Page {
	isPublished := false
	page := &model.Page{NamespaceCode: namespaceCode, ProjectCode: projectCode, IsPublished: &isPublished}
	err := db.Create(page).Error
	assert.NoError(t, err)
	return page
}

func TestNewPageDraftRepository(t *testing.T) {
	db := setupPageDraftTestDB(t)
	repo := NewPageDraftRepository(db)
	assert.NotNil(t, repo)
}

func TestPageDraftRepository_GetTx(t *testing.T) {
	db := setupPageDraftTestDB(t)
	repo := NewPageDraftRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var drafts []model.PageDraft
	err := tx.Find(&drafts).Error
	assert.NoError(t, err)
}

func TestPageDraftRepository_GetQuery(t *testing.T) {
	db := setupPageDraftTestDB(t)
	repo := NewPageDraftRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var drafts []model.PageDraft
	err := query.Find(&drafts).Error
	assert.NoError(t, err)
}

func TestPageDraftRepository_FindByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		page := createTestPage(t, db, "test-ns", "test-proj")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewPage: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        "/path",
				Content:     "content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		db.Create(draft)

		result, err := repo.FindByID(ctx, draft.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, draft.ID, result.ID)
		assert.NotNil(t, result.OldPage)
	})

	t.Run("not found", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		result, err := repo.FindByID(ctx, 999)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPageDraftRepository_FindByIDWithProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		page := createTestPage(t, db, "test-ns", "test-proj")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		result, err := repo.FindByIDWithProject(ctx, "test-ns", "test-proj", draft.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, draft.ID, result.ID)
	})

	t.Run("wrong namespace", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		page := createTestPage(t, db, "test-ns", "test-proj")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		result, err := repo.FindByIDWithProject(ctx, "other-ns", "test-proj", draft.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("wrong project", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		page := createTestPage(t, db, "test-ns", "test-proj")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		result, err := repo.FindByIDWithProject(ctx, "test-ns", "other-proj", draft.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPageDraftRepository_FindByProject(t *testing.T) {
	t.Run("success returns drafts for project", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			page := createTestPage(t, db, "test-ns", "test-proj")
			db.Create(&model.PageDraft{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				OldPageID:     &page.ID,
				ChangeType:    model.DraftChangeTypeUpdate,
			})
		}

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Len(t, results, 3)
		for _, draft := range results {
			assert.Equal(t, "test-ns", draft.NamespaceCode)
			assert.Equal(t, "test-proj", draft.ProjectCode)
			assert.NotNil(t, draft.OldPage)
		}
	})

	t.Run("returns empty slice when no drafts", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("only returns drafts for specified project", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "proj-a", "Project A")
		createTestPageDraftProject(t, db, "test-ns", "proj-b", "Project B")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		for i := 0; i < 2; i++ {
			page := createTestPage(t, db, "test-ns", "proj-a")
			db.Create(&model.PageDraft{
				NamespaceCode: "test-ns",
				ProjectCode:   "proj-a",
				OldPageID:     &page.ID,
				ChangeType:    model.DraftChangeTypeUpdate,
			})
		}

		for i := 0; i < 3; i++ {
			page := createTestPage(t, db, "test-ns", "proj-b")
			db.Create(&model.PageDraft{
				NamespaceCode: "test-ns",
				ProjectCode:   "proj-b",
				OldPageID:     &page.ID,
				ChangeType:    model.DraftChangeTypeUpdate,
			})
		}

		results, err := repo.FindByProject(ctx, "test-ns", "proj-a")

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		for _, draft := range results {
			assert.Equal(t, "proj-a", draft.ProjectCode)
		}
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		sqlDB, _ := db.DB()
		sqlDB.Close()

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Nil(t, results)
	})
}

func TestPageDraftRepository_Create(t *testing.T) {
	db := setupPageDraftTestDB(t)
	createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	page := createTestPage(t, db, "test-ns", "test-proj")
	repo := NewPageDraftRepository(db)
	ctx := context.Background()

	draft := &model.PageDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldPageID:     &page.ID,
		ChangeType:    model.DraftChangeTypeCreate,
		NewPage: &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/new-path",
			Content:     "new content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		},
	}

	err := repo.Create(ctx, draft)

	assert.NoError(t, err)
	assert.NotZero(t, draft.ID)

	var found model.PageDraft
	db.First(&found, draft.ID)
	assert.Equal(t, "/new-path", found.NewPage.Path)
}

func TestPageDraftRepository_Update(t *testing.T) {
	db := setupPageDraftTestDB(t)
	createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	page := createTestPage(t, db, "test-ns", "test-proj")
	repo := NewPageDraftRepository(db)
	ctx := context.Background()

	draft := &model.PageDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldPageID:     &page.ID,
		ChangeType:    model.DraftChangeTypeUpdate,
		NewPage: &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/original",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		},
	}
	db.Create(draft)

	draft.NewPage.Path = "/updated"
	err := repo.Update(ctx, draft)

	assert.NoError(t, err)

	var found model.PageDraft
	db.First(&found, draft.ID)
	assert.Equal(t, "/updated", found.NewPage.Path)
}

func TestPageDraftRepository_Delete(t *testing.T) {
	db := setupPageDraftTestDB(t)
	createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	page := createTestPage(t, db, "test-ns", "test-proj")
	repo := NewPageDraftRepository(db)
	ctx := context.Background()

	draft := &model.PageDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldPageID:     &page.ID,
		ChangeType:    model.DraftChangeTypeUpdate,
	}
	db.Create(draft)

	err := repo.Delete(ctx, draft.ID)

	assert.NoError(t, err)

	var found model.PageDraft
	result := db.First(&found, draft.ID)
	assert.Error(t, result.Error)
}

func TestPageDraftRepository_Search(t *testing.T) {
	db := setupPageDraftTestDB(t)
	createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewPageDraftRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		page := createTestPage(t, db, "test-ns", "test-proj")
		db.Create(&model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		})
	}

	t.Run("search with nil query returns all", func(t *testing.T) {
		results, err := repo.Search(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, results, 5)
	})

	t.Run("search with custom query", func(t *testing.T) {
		query := db.Model(&model.PageDraft{}).Where("namespace_code = ?", "test-ns").Limit(2)
		results, err := repo.Search(ctx, query)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestPageDraftRepository_SearchPaginate(t *testing.T) {
	db := setupPageDraftTestDB(t)
	createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewPageDraftRepository(db)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		page := createTestPage(t, db, "test-ns", "test-proj")
		db.Create(&model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		})
	}

	tests := []struct {
		name      string
		limit     int
		offset    int
		wantCount int
		wantTotal int64
	}{
		{
			name:      "paginate with limit",
			limit:     5,
			offset:    0,
			wantCount: 5,
			wantTotal: 15,
		},
		{
			name:      "paginate with offset",
			limit:     5,
			offset:    10,
			wantCount: 5,
			wantTotal: 15,
		},
		{
			name:      "paginate without limit returns all",
			limit:     0,
			offset:    0,
			wantCount: 15,
			wantTotal: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, total, err := repo.SearchPaginate(ctx, nil, tt.limit, tt.offset)

			assert.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestPageDraftRepository_SearchPaginate_PreloadsOldPage(t *testing.T) {
	db := setupPageDraftTestDB(t)
	createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	page := createTestPage(t, db, "test-ns", "test-proj")
	repo := NewPageDraftRepository(db)
	ctx := context.Background()

	draft := &model.PageDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldPageID:     &page.ID,
		ChangeType:    model.DraftChangeTypeUpdate,
	}
	db.Create(draft)

	results, _, err := repo.SearchPaginate(ctx, nil, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NotNil(t, results[0].OldPage)
	assert.Equal(t, page.ID, results[0].OldPage.ID)
}

func TestPageDraftRepository_CheckPathAvailability(t *testing.T) {
	t.Run("path available when no conflicts", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		available, err := repo.CheckPathAvailability(ctx, "test-ns", "test-proj", "/new-path", nil, nil)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("path unavailable when exists in pages", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		page := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Page: &commonTypes.Page{
				Path:        "/existing-path",
				Content:     "content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		db.Create(page)

		available, err := repo.CheckPathAvailability(ctx, "test-ns", "test-proj", "/existing-path", nil, nil)

		assert.NoError(t, err)
		assert.False(t, available)
	})

	t.Run("path unavailable when exists in page_drafts", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    model.DraftChangeTypeCreate,
			NewPage: &commonTypes.Page{
				Path:        "/draft-path",
				Content:     "content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		db.Create(draft)

		available, err := repo.CheckPathAvailability(ctx, "test-ns", "test-proj", "/draft-path", nil, nil)

		assert.NoError(t, err)
		assert.False(t, available)
	})

	t.Run("path available when excluded page matches", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		page := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Page: &commonTypes.Page{
				Path:        "/my-path",
				Content:     "content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		db.Create(page)

		available, err := repo.CheckPathAvailability(ctx, "test-ns", "test-proj", "/my-path", &page.ID, nil)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("path available when excluded draft matches", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    model.DraftChangeTypeCreate,
			NewPage: &commonTypes.Page{
				Path:        "/my-draft-path",
				Content:     "content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		db.Create(draft)

		available, err := repo.CheckPathAvailability(ctx, "test-ns", "test-proj", "/my-draft-path", nil, &draft.ID)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("path available when draft is DELETE type", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		page := createTestPage(t, db, "test-ns", "test-proj")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			ChangeType:    model.DraftChangeTypeDelete,
			NewPage: &commonTypes.Page{
				Path:        "/delete-path",
				Content:     "content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		db.Create(draft)

		available, err := repo.CheckPathAvailability(ctx, "test-ns", "test-proj", "/delete-path", nil, nil)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("path available in different project", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		createTestPageDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageDraftProject(t, db, "test-ns", "proj-a", "Project A")
		createTestPageDraftProject(t, db, "test-ns", "proj-b", "Project B")
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		page := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "proj-a",
			Page: &commonTypes.Page{
				Path:        "/same-path",
				Content:     "content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		db.Create(page)

		available, err := repo.CheckPathAvailability(ctx, "test-ns", "proj-b", "/same-path", nil, nil)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := setupPageDraftTestDB(t)
		repo := NewPageDraftRepository(db)
		ctx := context.Background()

		sqlDB, _ := db.DB()
		sqlDB.Close()

		available, err := repo.CheckPathAvailability(ctx, "test-ns", "test-proj", "/path", nil, nil)

		assert.Error(t, err)
		assert.False(t, available)
	})
}