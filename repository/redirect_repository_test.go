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

func setupRedirectTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
	assert.NoError(t, err)

	return db
}

func createTestRedirectNamespace(t *testing.T, db *gorm.DB, code, name string) *model.Namespace {
	ns := &model.Namespace{
		NamespaceCode: code,
		Name:          name,
	}
	err := db.Create(ns).Error
	assert.NoError(t, err)
	return ns
}

func createTestRedirectProject(t *testing.T, db *gorm.DB, namespaceCode, projectCode, name string) *model.Project {
	proj := &model.Project{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		Name:          name,
	}
	err := db.Create(proj).Error
	assert.NoError(t, err)
	return proj
}

func TestNewRedirectRepository(t *testing.T) {
	db := setupRedirectTestDB(t)
	repo := NewRedirectRepository(db)

	assert.NotNil(t, repo)
}

func TestRedirectRepository_GetTx(t *testing.T) {
	db := setupRedirectTestDB(t)
	repo := NewRedirectRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var redirects []model.Redirect
	err := tx.Find(&redirects).Error
	assert.NoError(t, err)
}

func TestRedirectRepository_GetQuery(t *testing.T) {
	db := setupRedirectTestDB(t)
	repo := NewRedirectRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var redirects []model.Redirect
	err := query.Find(&redirects).Error
	assert.NoError(t, err)
}

func TestRedirectRepository_FindByID(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(db *gorm.DB) int64
		namespaceCode string
		projectCode   string
		wantErr       bool
	}{
		{
			name: "find existing redirect",
			setupFunc: func(db *gorm.DB) int64 {
				redirect := &model.Redirect{
					NamespaceCode: "test-ns",
					ProjectCode:   "test-proj",
					IsPublished:   boolPtr(true),
					Redirect: &commonTypes.Redirect{
						Type:   commonTypes.RedirectTypeBasic,
						Source: "/old",
						Target: "/new",
						Status: commonTypes.RedirectStatusMovedPermanent,
					},
				}
				db.Create(redirect)
				return redirect.ID
			},
			namespaceCode: "test-ns",
			projectCode:   "test-proj",
			wantErr:       false,
		},
		{
			name: "redirect not found",
			setupFunc: func(db *gorm.DB) int64 {
				return 999
			},
			namespaceCode: "test-ns",
			projectCode:   "test-proj",
			wantErr:       true,
		},
		{
			name: "redirect wrong namespace",
			setupFunc: func(db *gorm.DB) int64 {
				redirect := &model.Redirect{
					NamespaceCode: "test-ns",
					ProjectCode:   "test-proj",
					IsPublished:   boolPtr(true),
				}
				db.Create(redirect)
				return redirect.ID
			},
			namespaceCode: "other-ns",
			projectCode:   "test-proj",
			wantErr:       true,
		},
		{
			name: "redirect wrong project",
			setupFunc: func(db *gorm.DB) int64 {
				redirect := &model.Redirect{
					NamespaceCode: "test-ns",
					ProjectCode:   "test-proj",
					IsPublished:   boolPtr(true),
				}
				db.Create(redirect)
				return redirect.ID
			},
			namespaceCode: "test-ns",
			projectCode:   "other-proj",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupRedirectTestDB(t)
			createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
			createTestRedirectNamespace(t, db, "other-ns", "Other Namespace")
			createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
			createTestRedirectProject(t, db, "other-ns", "other-proj", "Other Project")
			repo := NewRedirectRepository(db)
			ctx := context.Background()

			redirectID := tt.setupFunc(db)

			result, err := repo.FindByID(ctx, tt.namespaceCode, tt.projectCode, redirectID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, redirectID, result.ID)
				assert.Equal(t, tt.namespaceCode, result.NamespaceCode)
				assert.Equal(t, tt.projectCode, result.ProjectCode)
			}
		})
	}
}

func TestRedirectRepository_FindByID_PreloadsRedirectDraft(t *testing.T) {
	db := setupRedirectTestDB(t)
	createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
	createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewRedirectRepository(db)
	ctx := context.Background()

	redirect := &model.Redirect{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		IsPublished:   boolPtr(false),
	}
	db.Create(redirect)

	draft := &model.RedirectDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldRedirectID: &redirect.ID,
		NewRedirect: &commonTypes.Redirect{
			Type:   commonTypes.RedirectTypeBasic,
			Source: "/draft-source",
			Target: "/draft-target",
			Status: commonTypes.RedirectStatusMovedPermanent,
		},
	}
	db.Create(draft)

	result, err := repo.FindByID(ctx, "test-ns", "test-proj", redirect.ID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.RedirectDraft)
	assert.Equal(t, "/draft-source", result.RedirectDraft.NewRedirect.Source)
}

func TestRedirectRepository_FindByProject(t *testing.T) {
	t.Run("success returns redirects for project", func(t *testing.T) {
		db := setupRedirectTestDB(t)
		createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
		createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		// Create multiple redirects for the project
		for i := 0; i < 3; i++ {
			db.Create(&model.Redirect{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(true),
			})
		}

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Len(t, results, 3)
		for _, redirect := range results {
			assert.Equal(t, "test-ns", redirect.NamespaceCode)
			assert.Equal(t, "test-proj", redirect.ProjectCode)
		}
	})

	t.Run("returns empty slice when no redirects", func(t *testing.T) {
		db := setupRedirectTestDB(t)
		createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
		createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("only returns redirects for specified project", func(t *testing.T) {
		db := setupRedirectTestDB(t)
		createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
		createTestRedirectProject(t, db, "test-ns", "proj-a", "Project A")
		createTestRedirectProject(t, db, "test-ns", "proj-b", "Project B")
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		// Create redirects for proj-a
		for i := 0; i < 2; i++ {
			db.Create(&model.Redirect{
				NamespaceCode: "test-ns",
				ProjectCode:   "proj-a",
				IsPublished:   boolPtr(true),
			})
		}

		// Create redirects for proj-b
		for i := 0; i < 3; i++ {
			db.Create(&model.Redirect{
				NamespaceCode: "test-ns",
				ProjectCode:   "proj-b",
				IsPublished:   boolPtr(true),
			})
		}

		results, err := repo.FindByProject(ctx, "test-ns", "proj-a")

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		for _, redirect := range results {
			assert.Equal(t, "proj-a", redirect.ProjectCode)
		}
	})

	t.Run("preloads redirect draft", func(t *testing.T) {
		db := setupRedirectTestDB(t)
		createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
		createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		redirect := &model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(false),
		}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			OldRedirectID: &redirect.ID,
			NewRedirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/draft-source",
				Target: "/draft-target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.NotNil(t, results[0].RedirectDraft)
		assert.Equal(t, "/draft-source", results[0].RedirectDraft.NewRedirect.Source)
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		db := setupRedirectTestDB(t)
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		// Close the database to trigger an error
		sqlDB, _ := db.DB()
		sqlDB.Close()

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Nil(t, results)
	})
}

func TestRedirectRepository_FindByProjectPublished(t *testing.T) {
	t.Run("returns only published redirects", func(t *testing.T) {
		db := setupRedirectTestDB(t)
		createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
		createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		// Create published redirects
		for i := 0; i < 3; i++ {
			db.Create(&model.Redirect{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(true),
			})
		}
		// Create unpublished redirects
		for i := 0; i < 2; i++ {
			db.Create(&model.Redirect{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(false),
			})
		}

		results, total, err := repo.FindByProjectPublished(ctx, "test-ns", "test-proj", 0, 0)

		assert.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, int64(3), total)
		for _, redirect := range results {
			assert.True(t, *redirect.IsPublished)
		}
	})

	t.Run("pagination with limit", func(t *testing.T) {
		db := setupRedirectTestDB(t)
		createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
		createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		for i := 0; i < 10; i++ {
			db.Create(&model.Redirect{
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
		db := setupRedirectTestDB(t)
		createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
		createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		for i := 0; i < 10; i++ {
			db.Create(&model.Redirect{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				IsPublished:   boolPtr(true),
			})
		}

		results, total, err := repo.FindByProjectPublished(ctx, "test-ns", "test-proj", 5, 7)

		assert.NoError(t, err)
		assert.Len(t, results, 3) // Only 3 remaining after offset 7
		assert.Equal(t, int64(10), total)
	})

	t.Run("returns empty when no published redirects", func(t *testing.T) {
		db := setupRedirectTestDB(t)
		createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
		createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		// Create only unpublished redirects
		for i := 0; i < 3; i++ {
			db.Create(&model.Redirect{
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
		db := setupRedirectTestDB(t)
		createTestRedirectNamespace(t, db, "ns-a", "Namespace A")
		createTestRedirectNamespace(t, db, "ns-b", "Namespace B")
		createTestRedirectProject(t, db, "ns-a", "proj-a", "Project A")
		createTestRedirectProject(t, db, "ns-b", "proj-b", "Project B")
		repo := NewRedirectRepository(db)
		ctx := context.Background()

		// Create published redirects in ns-a/proj-a
		for i := 0; i < 5; i++ {
			db.Create(&model.Redirect{
				NamespaceCode: "ns-a",
				ProjectCode:   "proj-a",
				IsPublished:   boolPtr(true),
			})
		}
		// Create published redirects in ns-b/proj-b
		for i := 0; i < 3; i++ {
			db.Create(&model.Redirect{
				NamespaceCode: "ns-b",
				ProjectCode:   "proj-b",
				IsPublished:   boolPtr(true),
			})
		}

		results, total, err := repo.FindByProjectPublished(ctx, "ns-a", "proj-a", 0, 0)

		assert.NoError(t, err)
		assert.Len(t, results, 5)
		assert.Equal(t, int64(5), total)
		for _, redirect := range results {
			assert.Equal(t, "ns-a", redirect.NamespaceCode)
			assert.Equal(t, "proj-a", redirect.ProjectCode)
		}
	})
}

func TestRedirectRepository_Search(t *testing.T) {
	db := setupRedirectTestDB(t)
	createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
	createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewRedirectRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		db.Create(&model.Redirect{
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
		query := db.Model(&model.Redirect{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "test-proj").Limit(2)
		results, err := repo.Search(ctx, query)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestRedirectRepository_SearchPaginate(t *testing.T) {
	db := setupRedirectTestDB(t)
	createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
	createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewRedirectRepository(db)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		db.Create(&model.Redirect{
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

func TestRedirectRepository_SearchPaginate_WithFilter(t *testing.T) {
	db := setupRedirectTestDB(t)
	createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
	createTestRedirectNamespace(t, db, "other-ns", "Other Namespace")
	createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
	createTestRedirectProject(t, db, "other-ns", "other-proj", "Other Project")
	repo := NewRedirectRepository(db)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		db.Create(&model.Redirect{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			IsPublished:   boolPtr(true),
		})
	}
	for i := 0; i < 5; i++ {
		db.Create(&model.Redirect{
			NamespaceCode: "other-ns",
			ProjectCode:   "other-proj",
			IsPublished:   boolPtr(true),
		})
	}

	query := db.Model(&model.Redirect{}).Where("namespace_code = ? AND project_code = ?", "test-ns", "test-proj")
	results, total, err := repo.SearchPaginate(ctx, query, 5, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 5)
	assert.Equal(t, int64(10), total)
}

func TestRedirectRepository_SearchPaginate_PreloadsRedirectDraft(t *testing.T) {
	db := setupRedirectTestDB(t)
	createTestRedirectNamespace(t, db, "test-ns", "Test Namespace")
	createTestRedirectProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewRedirectRepository(db)
	ctx := context.Background()

	redirect := &model.Redirect{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		IsPublished:   boolPtr(false),
	}
	db.Create(redirect)

	draft := &model.RedirectDraft{
		NamespaceCode: "test-ns",
		ProjectCode:   "test-proj",
		OldRedirectID: &redirect.ID,
		NewRedirect: &commonTypes.Redirect{
			Type:   commonTypes.RedirectTypeBasic,
			Source: "/source",
			Target: "/target",
			Status: commonTypes.RedirectStatusMovedPermanent,
		},
	}
	db.Create(draft)

	results, _, err := repo.SearchPaginate(ctx, nil, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NotNil(t, results[0].RedirectDraft)
	assert.Equal(t, "/source", results[0].RedirectDraft.NewRedirect.Source)
}
