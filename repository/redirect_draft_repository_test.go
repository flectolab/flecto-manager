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

func setupRedirectDraftTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
	assert.NoError(t, err)

	return db
}

func createTestDraftNamespace(t *testing.T, db *gorm.DB, code, name string) *model.Namespace {
	ns := &model.Namespace{NamespaceCode: code, Name: name}
	err := db.Create(ns).Error
	assert.NoError(t, err)
	return ns
}

func createTestDraftProject(t *testing.T, db *gorm.DB, namespaceCode, projectCode, name string) *model.Project {
	proj := &model.Project{NamespaceCode: namespaceCode, ProjectCode: projectCode, Name: name}
	err := db.Create(proj).Error
	assert.NoError(t, err)
	return proj
}

func createTestRedirect(t *testing.T, db *gorm.DB, namespaceCode, projectCode string) *model.Redirect {
	isPublished := false
	redirect := &model.Redirect{NamespaceCode: namespaceCode, ProjectCode: projectCode, IsPublished: &isPublished}
	err := db.Create(redirect).Error
	assert.NoError(t, err)
	return redirect
}

func TestNewRedirectDraftRepository(t *testing.T) {
	db := setupRedirectDraftTestDB(t)
	repo := NewRedirectDraftRepository(db)
	assert.NotNil(t, repo)
}

func TestRedirectDraftRepository_GetTx(t *testing.T) {
	db := setupRedirectDraftTestDB(t)
	repo := NewRedirectDraftRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var drafts []model.RedirectDraft
	err := tx.Find(&drafts).Error
	assert.NoError(t, err)
}

func TestRedirectDraftRepository_GetQuery(t *testing.T) {
	db := setupRedirectDraftTestDB(t)
	repo := NewRedirectDraftRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var drafts []model.RedirectDraft
	err := query.Find(&drafts).Error
	assert.NoError(t, err)
}

func TestRedirectDraftRepository_FindByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		redirect := createTestRedirect(t, db, "test-ns", "test-proj")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		result, err := repo.FindByID(ctx, draft.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, draft.ID, result.ID)
		assert.NotNil(t, result.OldRedirect)
	})

	t.Run("not found", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		result, err := repo.FindByID(ctx, 999)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestRedirectDraftRepository_FindByIDWithProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		redirect := createTestRedirect(t, db, "test-ns", "test-proj")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		result, err := repo.FindByIDWithProject(ctx, "test-ns", "test-proj", draft.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, draft.ID, result.ID)
	})

	t.Run("wrong namespace", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		redirect := createTestRedirect(t, db, "test-ns", "test-proj")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		result, err := repo.FindByIDWithProject(ctx, "other-ns", "test-proj", draft.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("wrong project", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		redirect := createTestRedirect(t, db, "test-ns", "test-proj")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		}
		db.Create(draft)

		result, err := repo.FindByIDWithProject(ctx, "test-ns", "other-proj", draft.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestRedirectDraftRepository_FindByProject(t *testing.T) {
	t.Run("success returns drafts for project", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		// Create multiple drafts for the project
		for i := 0; i < 3; i++ {
			redirect := createTestRedirect(t, db, "test-ns", "test-proj")
			db.Create(&model.RedirectDraft{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				OldRedirectID: &redirect.ID,
				ChangeType:    model.DraftChangeTypeUpdate,
			})
		}

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Len(t, results, 3)
		for _, draft := range results {
			assert.Equal(t, "test-ns", draft.NamespaceCode)
			assert.Equal(t, "test-proj", draft.ProjectCode)
			assert.NotNil(t, draft.OldRedirect)
		}
	})

	t.Run("returns empty slice when no drafts", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("only returns drafts for specified project", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "proj-a", "Project A")
		createTestDraftProject(t, db, "test-ns", "proj-b", "Project B")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		// Create drafts for proj-a
		for i := 0; i < 2; i++ {
			redirect := createTestRedirect(t, db, "test-ns", "proj-a")
			db.Create(&model.RedirectDraft{
				NamespaceCode: "test-ns",
				ProjectCode:   "proj-a",
				OldRedirectID: &redirect.ID,
				ChangeType:    model.DraftChangeTypeUpdate,
			})
		}

		// Create drafts for proj-b
		for i := 0; i < 3; i++ {
			redirect := createTestRedirect(t, db, "test-ns", "proj-b")
			db.Create(&model.RedirectDraft{
				NamespaceCode: "test-ns",
				ProjectCode:   "proj-b",
				OldRedirectID: &redirect.ID,
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
		db := setupRedirectDraftTestDB(t)
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		// Close the database to trigger an error
		sqlDB, _ := db.DB()
		sqlDB.Close()

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Nil(t, results)
	})
}

func TestRedirectDraftRepository_Create(t *testing.T) {
	db := setupRedirectDraftTestDB(t)
	createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	redirect := createTestRedirect(t, db, "test-ns", "test-proj")
	repo := NewRedirectDraftRepository(db)
	ctx := context.Background()

	draft := &model.RedirectDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldRedirectID: &redirect.ID,
		ChangeType:    model.DraftChangeTypeCreate,
		NewRedirect: &commonTypes.Redirect{
			Type:   commonTypes.RedirectTypeBasic,
			Source: "/new-source",
			Target: "/new-target",
			Status: commonTypes.RedirectStatusMovedPermanent,
		},
	}

	err := repo.Create(ctx, draft)

	assert.NoError(t, err)
	assert.NotZero(t, draft.ID)

	var found model.RedirectDraft
	db.First(&found, draft.ID)
	assert.Equal(t, "/new-source", found.NewRedirect.Source)
}

func TestRedirectDraftRepository_Update(t *testing.T) {
	db := setupRedirectDraftTestDB(t)
	createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	redirect := createTestRedirect(t, db, "test-ns", "test-proj")
	repo := NewRedirectDraftRepository(db)
	ctx := context.Background()

	draft := &model.RedirectDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldRedirectID: &redirect.ID,
		ChangeType:    model.DraftChangeTypeUpdate,
		NewRedirect: &commonTypes.Redirect{
			Type:   commonTypes.RedirectTypeBasic,
			Source: "/original",
			Target: "/target",
			Status: commonTypes.RedirectStatusMovedPermanent,
		},
	}
	db.Create(draft)

	draft.NewRedirect.Source = "/updated"
	err := repo.Update(ctx, draft)

	assert.NoError(t, err)

	var found model.RedirectDraft
	db.First(&found, draft.ID)
	assert.Equal(t, "/updated", found.NewRedirect.Source)
}

func TestRedirectDraftRepository_Delete(t *testing.T) {
	db := setupRedirectDraftTestDB(t)
	createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	redirect := createTestRedirect(t, db, "test-ns", "test-proj")
	repo := NewRedirectDraftRepository(db)
	ctx := context.Background()

	draft := &model.RedirectDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldRedirectID: &redirect.ID,
		ChangeType:    model.DraftChangeTypeUpdate,
	}
	db.Create(draft)

	err := repo.Delete(ctx, draft.ID)

	assert.NoError(t, err)

	var found model.RedirectDraft
	result := db.First(&found, draft.ID)
	assert.Error(t, result.Error)
}

func TestRedirectDraftRepository_Search(t *testing.T) {
	db := setupRedirectDraftTestDB(t)
	createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewRedirectDraftRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		redirect := createTestRedirect(t, db, "test-ns", "test-proj")
		db.Create(&model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
		})
	}

	t.Run("search with nil query returns all", func(t *testing.T) {
		results, err := repo.Search(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, results, 5)
	})

	t.Run("search with custom query", func(t *testing.T) {
		query := db.Model(&model.RedirectDraft{}).Where("namespace_code = ?", "test-ns").Limit(2)
		results, err := repo.Search(ctx, query)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestRedirectDraftRepository_SearchPaginate(t *testing.T) {
	db := setupRedirectDraftTestDB(t)
	createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewRedirectDraftRepository(db)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		redirect := createTestRedirect(t, db, "test-ns", "test-proj")
		db.Create(&model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
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

func TestRedirectDraftRepository_SearchPaginate_PreloadsOldRedirect(t *testing.T) {
	db := setupRedirectDraftTestDB(t)
	createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
	createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
	redirect := createTestRedirect(t, db, "test-ns", "test-proj")
	repo := NewRedirectDraftRepository(db)
	ctx := context.Background()

	draft := &model.RedirectDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldRedirectID: &redirect.ID,
		ChangeType:    model.DraftChangeTypeUpdate,
	}
	db.Create(draft)

	results, _, err := repo.SearchPaginate(ctx, nil, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NotNil(t, results[0].OldRedirect)
	assert.Equal(t, redirect.ID, results[0].OldRedirect.ID)
}

func TestRedirectDraftRepository_CheckSourceAvailability(t *testing.T) {
	t.Run("source available when no conflicts", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		available, err := repo.CheckSourceAvailability(ctx, "test-ns", "test-proj", "/new-source", nil, nil)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("source unavailable when exists in redirects", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		// Create a redirect with the source
		redirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Redirect: &commonTypes.Redirect{
				Source: "/existing-source",
				Target: "/target",
			},
		}
		db.Create(redirect)

		available, err := repo.CheckSourceAvailability(ctx, "test-ns", "test-proj", "/existing-source", nil, nil)

		assert.NoError(t, err)
		assert.False(t, available)
	})

	t.Run("source unavailable when exists in redirect_drafts", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		// Create a draft with the source
		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    model.DraftChangeTypeCreate,
			NewRedirect: &commonTypes.Redirect{
				Source: "/draft-source",
				Target: "/target",
			},
		}
		db.Create(draft)

		available, err := repo.CheckSourceAvailability(ctx, "test-ns", "test-proj", "/draft-source", nil, nil)

		assert.NoError(t, err)
		assert.False(t, available)
	})

	t.Run("source available when excluded redirect matches", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		redirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Redirect: &commonTypes.Redirect{
				Source: "/my-source",
				Target: "/target",
			},
		}
		db.Create(redirect)

		// Exclude the redirect that has this source
		available, err := repo.CheckSourceAvailability(ctx, "test-ns", "test-proj", "/my-source", &redirect.ID, nil)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("source available when excluded draft matches", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			ChangeType:    model.DraftChangeTypeCreate,
			NewRedirect: &commonTypes.Redirect{
				Source: "/my-draft-source",
				Target: "/target",
			},
		}
		db.Create(draft)

		// Exclude the draft that has this source
		available, err := repo.CheckSourceAvailability(ctx, "test-ns", "test-proj", "/my-draft-source", nil, &draft.ID)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("source available when draft is DELETE type", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "test-proj", "Test Project")
		redirect := createTestRedirect(t, db, "test-ns", "test-proj")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		// Create a DELETE draft (should not block the source)
		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeDelete,
			NewRedirect: &commonTypes.Redirect{
				Source: "/delete-source",
				Target: "/target",
			},
		}
		db.Create(draft)

		available, err := repo.CheckSourceAvailability(ctx, "test-ns", "test-proj", "/delete-source", nil, nil)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("source available in different project", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		createTestDraftNamespace(t, db, "test-ns", "Test Namespace")
		createTestDraftProject(t, db, "test-ns", "proj-a", "Project A")
		createTestDraftProject(t, db, "test-ns", "proj-b", "Project B")
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		// Create redirect in proj-a
		redirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "proj-a",
			Redirect: &commonTypes.Redirect{
				Source: "/same-source",
				Target: "/target",
			},
		}
		db.Create(redirect)

		// Check availability in proj-b (should be available)
		available, err := repo.CheckSourceAvailability(ctx, "test-ns", "proj-b", "/same-source", nil, nil)

		assert.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := setupRedirectDraftTestDB(t)
		repo := NewRedirectDraftRepository(db)
		ctx := context.Background()

		sqlDB, _ := db.DB()
		sqlDB.Close()

		available, err := repo.CheckSourceAvailability(ctx, "test-ns", "test-proj", "/source", nil, nil)

		assert.Error(t, err)
		assert.False(t, available)
	})
}
