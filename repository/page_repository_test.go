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

func setupPageTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Page{}, &model.PageDraft{})
	assert.NoError(t, err)

	return db
}

func createTestPageNamespace(t *testing.T, db *gorm.DB, code, name string) *model.Namespace {
	ns := &model.Namespace{
		NamespaceCode: code,
		Name:          name,
	}
	err := db.Create(ns).Error
	assert.NoError(t, err)
	return ns
}

func createTestPageProject(t *testing.T, db *gorm.DB, namespaceCode, projectCode, name string) *model.Project {
	proj := &model.Project{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		Name:          name,
	}
	err := db.Create(proj).Error
	assert.NoError(t, err)
	return proj
}

func TestNewPageRepository(t *testing.T) {
	db := setupPageTestDB(t)
	repo := NewPageRepository(db)

	assert.NotNil(t, repo)
}

func TestPageRepository_GetTx(t *testing.T) {
	db := setupPageTestDB(t)
	repo := NewPageRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var pages []model.Page
	err := tx.Find(&pages).Error
	assert.NoError(t, err)
}

func TestPageRepository_GetQuery(t *testing.T) {
	db := setupPageTestDB(t)
	repo := NewPageRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var pages []model.Page
	err := query.Find(&pages).Error
	assert.NoError(t, err)
}

func TestPageRepository_FindByID(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(db *gorm.DB) int64
		namespaceCode string
		projectCode   string
		wantErr       bool
	}{
		{
			name: "find existing page",
			setupFunc: func(db *gorm.DB) int64 {
				page := &model.Page{
					NamespaceCode: "test-ns",
					ProjectCode:   "test-proj",
					IsPublished:   boolPtr(true),
					Page: &commonTypes.Page{
						Type:        commonTypes.PageTypeBasic,
						Path:        "/robots.txt",
						Content:     "User-agent: *",
						ContentType: commonTypes.PageContentTypeTextPlain,
					},
				}
				db.Create(page)
				return page.ID
			},
			namespaceCode: "test-ns",
			projectCode:   "test-proj",
			wantErr:       false,
		},
		{
			name: "page not found",
			setupFunc: func(db *gorm.DB) int64 {
				return 999
			},
			namespaceCode: "test-ns",
			projectCode:   "test-proj",
			wantErr:       true,
		},
		{
			name: "page wrong namespace",
			setupFunc: func(db *gorm.DB) int64 {
				page := &model.Page{
					NamespaceCode: "test-ns",
					ProjectCode:   "test-proj",
					IsPublished:   boolPtr(true),
				}
				db.Create(page)
				return page.ID
			},
			namespaceCode: "other-ns",
			projectCode:   "test-proj",
			wantErr:       true,
		},
		{
			name: "page wrong project",
			setupFunc: func(db *gorm.DB) int64 {
				page := &model.Page{
					NamespaceCode: "test-ns",
					ProjectCode:   "test-proj",
					IsPublished:   boolPtr(true),
				}
				db.Create(page)
				return page.ID
			},
			namespaceCode: "test-ns",
			projectCode:   "other-proj",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupPageTestDB(t)
			createTestPageNamespace(t, db, "test-ns", "Test Namespace")
			createTestPageNamespace(t, db, "other-ns", "Other Namespace")
			createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
			createTestPageProject(t, db, "other-ns", "other-proj", "Other Project")
			repo := NewPageRepository(db)
			ctx := context.Background()

			pageID := tt.setupFunc(db)

			result, err := repo.FindByID(ctx, tt.namespaceCode, tt.projectCode, pageID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, pageID, result.ID)
				assert.Equal(t, tt.namespaceCode, result.NamespaceCode)
				assert.Equal(t, tt.projectCode, result.ProjectCode)
			}
		})
	}
}

func TestPageRepository_FindByID_PreloadsPageDraft(t *testing.T) {
	db := setupPageTestDB(t)
	createTestPageNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewPageRepository(db)
	ctx := context.Background()

	page := &model.Page{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		IsPublished:   boolPtr(false),
	}
	db.Create(page)

	draft := &model.PageDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldPageID:     &page.ID,
		NewPage: &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/draft-path",
			Content:     "draft content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		},
	}
	db.Create(draft)

	result, err := repo.FindByID(ctx, "test-ns", "test-proj", page.ID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.PageDraft)
	assert.Equal(t, "/draft-path", result.PageDraft.NewPage.Path)
}

func TestPageRepository_FindByProject(t *testing.T) {
	t.Run("success returns pages for project", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			db.Create(&model.Page{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(true),
			})
		}

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Len(t, results, 3)
		for _, page := range results {
			assert.Equal(t, "test-ns", page.NamespaceCode)
			assert.Equal(t, "test-proj", page.ProjectCode)
		}
	})

	t.Run("returns empty slice when no pages", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("only returns pages for specified project", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "proj-a", "Project A")
		createTestPageProject(t, db, "test-ns", "proj-b", "Project B")
		repo := NewPageRepository(db)
		ctx := context.Background()

		for i := 0; i < 2; i++ {
			db.Create(&model.Page{
				NamespaceCode: "test-ns",
				ProjectCode:   "proj-a",
				IsPublished:   boolPtr(true),
			})
		}

		for i := 0; i < 3; i++ {
			db.Create(&model.Page{
				NamespaceCode: "test-ns",
				ProjectCode:   "proj-b",
				IsPublished:   boolPtr(true),
			})
		}

		results, err := repo.FindByProject(ctx, "test-ns", "proj-a")

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		for _, page := range results {
			assert.Equal(t, "proj-a", page.ProjectCode)
		}
	})

	t.Run("preloads page draft", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		page := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(false),
		}
		db.Create(page)

		draft := &model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &page.ID,
			NewPage: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        "/draft-path",
				Content:     "draft content",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
		}
		db.Create(draft)

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.NotNil(t, results[0].PageDraft)
		assert.Equal(t, "/draft-path", results[0].PageDraft.NewPage.Path)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := setupPageTestDB(t)
		repo := NewPageRepository(db)
		ctx := context.Background()

		sqlDB, _ := db.DB()
		sqlDB.Close()

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Nil(t, results)
	})
}

func TestPageRepository_FindByProjectPublished(t *testing.T) {
	t.Run("returns only published pages", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			db.Create(&model.Page{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(true),
			})
		}
		for i := 0; i < 2; i++ {
			db.Create(&model.Page{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(false),
			})
		}

		results, total, err := repo.FindByProjectPublished(ctx, "test-ns", "test-proj", 0, 0)

		assert.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, int64(3), total)
		for _, page := range results {
			assert.True(t, *page.IsPublished)
		}
	})

	t.Run("pagination with limit", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		for i := 0; i < 10; i++ {
			db.Create(&model.Page{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(true),
			})
		}

		results, total, err := repo.FindByProjectPublished(ctx, "test-ns", "test-proj", 5, 0)

		assert.NoError(t, err)
		assert.Len(t, results, 5)
		assert.Equal(t, int64(10), total)
	})

	t.Run("pagination with offset", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		for i := 0; i < 10; i++ {
			db.Create(&model.Page{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(true),
			})
		}

		results, total, err := repo.FindByProjectPublished(ctx, "test-ns", "test-proj", 5, 7)

		assert.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, int64(10), total)
	})

	t.Run("returns empty when no published pages", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			db.Create(&model.Page{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(false),
			})
		}

		results, total, err := repo.FindByProjectPublished(ctx, "test-ns", "test-proj", 0, 0)

		assert.NoError(t, err)
		assert.Empty(t, results)
		assert.Equal(t, int64(0), total)
	})

	t.Run("filters by namespace and project", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "ns-a", "Namespace A")
		createTestPageNamespace(t, db, "ns-b", "Namespace B")
		createTestPageProject(t, db, "ns-a", "proj-a", "Project A")
		createTestPageProject(t, db, "ns-b", "proj-b", "Project B")
		repo := NewPageRepository(db)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			db.Create(&model.Page{
				NamespaceCode: "ns-a",
				ProjectCode:   "proj-a",
				IsPublished:   boolPtr(true),
			})
		}
		for i := 0; i < 3; i++ {
			db.Create(&model.Page{
				NamespaceCode: "ns-b",
				ProjectCode:   "proj-b",
				IsPublished:   boolPtr(true),
			})
		}

		results, total, err := repo.FindByProjectPublished(ctx, "ns-a", "proj-a", 0, 0)

		assert.NoError(t, err)
		assert.Len(t, results, 5)
		assert.Equal(t, int64(5), total)
		for _, page := range results {
			assert.Equal(t, "ns-a", page.NamespaceCode)
			assert.Equal(t, "proj-a", page.ProjectCode)
		}
	})
}

func TestPageRepository_Search(t *testing.T) {
	db := setupPageTestDB(t)
	createTestPageNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewPageRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
		})
	}

	t.Run("search with nil query returns all", func(t *testing.T) {
		results, err := repo.Search(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, results, 5)
	})

	t.Run("search with custom query", func(t *testing.T) {
		query := db.Model(&model.Page{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "test-proj").Limit(2)
		results, err := repo.Search(ctx, query)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestPageRepository_SearchPaginate(t *testing.T) {
	db := setupPageTestDB(t)
	createTestPageNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewPageRepository(db)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
		})
	}

	tests := []struct {
		name      string
		query     *gorm.DB
		limit     int
		offset    int
		wantCount int
		wantTotal int64
	}{
		{
			name:      "paginate with limit",
			query:     nil,
			limit:     5,
			offset:    0,
			wantCount: 5,
			wantTotal: 15,
		},
		{
			name:      "paginate with offset",
			query:     nil,
			limit:     5,
			offset:    10,
			wantCount: 5,
			wantTotal: 15,
		},
		{
			name:      "paginate with offset beyond total",
			query:     nil,
			limit:     5,
			offset:    20,
			wantCount: 0,
			wantTotal: 15,
		},
		{
			name:      "paginate without limit returns all",
			query:     nil,
			limit:     0,
			offset:    0,
			wantCount: 15,
			wantTotal: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, total, err := repo.SearchPaginate(ctx, tt.query, tt.limit, tt.offset)

			assert.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestPageRepository_SearchPaginate_WithFilter(t *testing.T) {
	db := setupPageTestDB(t)
	createTestPageNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageNamespace(t, db, "other-ns", "Other Namespace")
	createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
	createTestPageProject(t, db, "other-ns", "other-proj", "Other Project")
	repo := NewPageRepository(db)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
		})
	}
	for i := 0; i < 5; i++ {
		db.Create(&model.Page{
			NamespaceCode: "other-ns",
			ProjectCode:   "other-proj",
			IsPublished:   boolPtr(true),
		})
	}

	query := db.Model(&model.Page{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "test-proj")
	results, total, err := repo.SearchPaginate(ctx, query, 5, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 5)
	assert.Equal(t, int64(10), total)
}

func TestPageRepository_SearchPaginate_PreloadsPageDraft(t *testing.T) {
	db := setupPageTestDB(t)
	createTestPageNamespace(t, db, "test-ns", "Test Namespace")
	createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewPageRepository(db)
	ctx := context.Background()

	page := &model.Page{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		IsPublished:   boolPtr(false),
	}
	db.Create(page)

	draft := &model.PageDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldPageID:     &page.ID,
		NewPage: &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/path",
			Content:     "content",
			ContentType: commonTypes.PageContentTypeTextPlain,
		},
	}
	db.Create(draft)

	results, _, err := repo.SearchPaginate(ctx, nil, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NotNil(t, results[0].PageDraft)
	assert.Equal(t, "/path", results[0].PageDraft.NewPage.Path)
}

func TestPageRepository_GetTotalContentSize(t *testing.T) {
	t.Run("returns zero for empty project", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		total, err := repo.GetTotalContentSize(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), total)
	})

	t.Run("sums content size of published pages without drafts", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		// Create published pages without drafts
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   100,
		})
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   200,
		})

		total, err := repo.GetTotalContentSize(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(300), total)
	})

	t.Run("excludes published pages that have drafts", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		// Published page without draft
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   100,
		})

		// Published page with draft (should be excluded from published sum)
		pageWithDraft := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   500, // This should NOT be counted
		}
		db.Create(pageWithDraft)

		// Draft for the page (this content size should be counted instead)
		db.Create(&model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &pageWithDraft.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   150, // This should be counted
		})

		total, err := repo.GetTotalContentSize(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(250), total) // 100 (published) + 150 (draft)
	})

	t.Run("includes CREATE and UPDATE drafts", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		// Page for CREATE draft
		pageCreate := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(false),
			ContentSize:   0,
		}
		db.Create(pageCreate)

		// CREATE draft
		db.Create(&model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &pageCreate.ID,
			ChangeType:    model.DraftChangeTypeCreate,
			ContentSize:   100,
		})

		// Page for UPDATE draft
		pageUpdate := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   50, // Should not be counted
		}
		db.Create(pageUpdate)

		// UPDATE draft
		db.Create(&model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &pageUpdate.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   200,
		})

		total, err := repo.GetTotalContentSize(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(300), total) // 100 (CREATE) + 200 (UPDATE)
	})

	t.Run("excludes DELETE drafts from total", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		// Published page without draft
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   100,
		})

		// Page to be deleted
		pageToDelete := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   500, // Should not be counted (has draft)
		}
		db.Create(pageToDelete)

		// DELETE draft (content size should not be counted)
		db.Create(&model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &pageToDelete.ID,
			ChangeType:    model.DraftChangeTypeDelete,
			ContentSize:   0, // DELETE drafts have no content
		})

		total, err := repo.GetTotalContentSize(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(100), total) // Only the published page without draft
	})

	t.Run("excludes unpublished pages from published sum", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		// Published page
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   100,
		})

		// Unpublished page (should not be counted in published sum)
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(false),
			ContentSize:   999,
		})

		total, err := repo.GetTotalContentSize(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(100), total)
	})

	t.Run("filters by namespace and project", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "ns-a", "Namespace A")
		createTestPageNamespace(t, db, "ns-b", "Namespace B")
		createTestPageProject(t, db, "ns-a", "proj-a", "Project A")
		createTestPageProject(t, db, "ns-b", "proj-b", "Project B")
		repo := NewPageRepository(db)
		ctx := context.Background()

		// Pages in project A
		db.Create(&model.Page{
			NamespaceCode: "ns-a",
			ProjectCode:   "proj-a",
			IsPublished:   boolPtr(true),
			ContentSize:   100,
		})

		// Pages in project B (should not be counted)
		db.Create(&model.Page{
			NamespaceCode: "ns-b",
			ProjectCode:   "proj-b",
			IsPublished:   boolPtr(true),
			ContentSize:   999,
		})

		total, err := repo.GetTotalContentSize(ctx, "ns-a", "proj-a")

		assert.NoError(t, err)
		assert.Equal(t, int64(100), total)
	})

	t.Run("complex scenario with mixed states", func(t *testing.T) {
		db := setupPageTestDB(t)
		createTestPageNamespace(t, db, "test-ns", "Test Namespace")
		createTestPageProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewPageRepository(db)
		ctx := context.Background()

		// Published page without draft (counted: 100)
		db.Create(&model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   100,
		})

		// Published page with UPDATE draft (page size 50 NOT counted, draft size 150 counted)
		pageUpdate := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   50,
		}
		db.Create(pageUpdate)
		db.Create(&model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &pageUpdate.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
			ContentSize:   150,
		})

		// New page with CREATE draft (counted: 200)
		pageCreate := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(false),
			ContentSize:   0,
		}
		db.Create(pageCreate)
		db.Create(&model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &pageCreate.ID,
			ChangeType:    model.DraftChangeTypeCreate,
			ContentSize:   200,
		})

		// Published page with DELETE draft (page has draft, so NOT counted; DELETE draft NOT counted)
		pageDelete := &model.Page{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
			ContentSize:   300,
		}
		db.Create(pageDelete)
		db.Create(&model.PageDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldPageID:     &pageDelete.ID,
			ChangeType:    model.DraftChangeTypeDelete,
			ContentSize:   0,
		})

		// Total should be: 100 + 150 + 200 = 450
		total, err := repo.GetTotalContentSize(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, int64(450), total)
	})
}