package repository

import (
	"context"
	"testing"

	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProjectTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{}, &model.Page{}, &model.PageDraft{})
	assert.NoError(t, err)

	return db
}

func createTestNamespace(t *testing.T, db *gorm.DB, code, name string) *model.Namespace {
	ns := &model.Namespace{
		NamespaceCode: code,
		Name:          name,
	}
	err := db.Create(ns).Error
	assert.NoError(t, err)
	return ns
}

func TestNewProjectRepository(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)

	assert.NotNil(t, repo)
}

func TestProjectRepository_GetTx(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var projects []model.Project
	err := tx.Find(&projects).Error
	assert.NoError(t, err)
}

func TestProjectRepository_GetQuery(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var projects []model.Project
	err := query.Find(&projects).Error
	assert.NoError(t, err)
}

func TestProjectRepository_Create(t *testing.T) {
	tests := []struct {
		name    string
		project *model.Project
		wantErr bool
	}{
		{
			name: "create valid project",
			project: &model.Project{
				ProjectCode:   "test-proj",
				NamespaceCode: "test-ns",
				Name:          "Test Project",
			},
			wantErr: false,
		},
		{
			name: "create project with version",
			project: &model.Project{
				ProjectCode:   "versioned-proj",
				NamespaceCode: "test-ns",
				Name:          "Versioned Project",
				Version:       5,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupProjectTestDB(t)
			createTestNamespace(t, db, "test-ns", "Test Namespace")
			repo := NewProjectRepository(db)
			ctx := context.Background()

			err := repo.Create(ctx, tt.project)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.project.ID)
			}
		})
	}
}

func TestProjectRepository_Create_DuplicateCode(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	proj1 := &model.Project{
		ProjectCode:   "duplicate-code",
		NamespaceCode: "test-ns",
		Name:          "First Project",
	}
	err := repo.Create(ctx, proj1)
	assert.NoError(t, err)

	proj2 := &model.Project{
		ProjectCode:   "duplicate-code",
		NamespaceCode: "test-ns",
		Name:          "Second Project",
	}
	err = repo.Create(ctx, proj2)
	assert.Error(t, err)
}

func TestProjectRepository_Create_SameCodeDifferentNamespace(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "ns-1", "Namespace 1")
	createTestNamespace(t, db, "ns-2", "Namespace 2")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	proj1 := &model.Project{
		ProjectCode:   "same-code",
		NamespaceCode: "ns-1",
		Name:          "Project in NS1",
	}
	err := repo.Create(ctx, proj1)
	assert.NoError(t, err)

	proj2 := &model.Project{
		ProjectCode:   "same-code",
		NamespaceCode: "ns-2",
		Name:          "Project in NS2",
	}
	err = repo.Create(ctx, proj2)
	assert.NoError(t, err)
}

func TestProjectRepository_Update(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	proj := &model.Project{
		ProjectCode:   "update-proj",
		NamespaceCode: "test-ns",
		Name:          "Original Name",
		Version:       1,
	}
	err := repo.Create(ctx, proj)
	assert.NoError(t, err)

	proj.Name = "Updated Name"
	proj.Version = 2
	err = repo.Update(ctx, proj)
	assert.NoError(t, err)

	found, err := repo.FindByCode(ctx, "test-ns", "update-proj")
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, 2, found.Version)
}

func TestProjectRepository_Delete(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(repo ProjectRepository, ctx context.Context)
		namespaceCode string
		projectCode   string
		wantErr       bool
	}{
		{
			name: "delete existing project",
			setupFunc: func(repo ProjectRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.Project{
					ProjectCode:   "to-delete",
					NamespaceCode: "test-ns",
					Name:          "To Delete",
				})
			},
			namespaceCode: "test-ns",
			projectCode:   "to-delete",
			wantErr:       false,
		},
		{
			name:          "delete non-existing project",
			setupFunc:     func(repo ProjectRepository, ctx context.Context) {},
			namespaceCode: "test-ns",
			projectCode:   "non-existing",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupProjectTestDB(t)
			createTestNamespace(t, db, "test-ns", "Test Namespace")
			repo := NewProjectRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			err := repo.Delete(ctx, tt.namespaceCode, tt.projectCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProjectRepository_DeleteByNamespaceCode(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "ns-to-delete", "Namespace To Delete")
	createTestNamespace(t, db, "ns-to-keep", "Namespace To Keep")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-1", NamespaceCode: "ns-to-delete", Name: "Project 1"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-2", NamespaceCode: "ns-to-delete", Name: "Project 2"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-3", NamespaceCode: "ns-to-keep", Name: "Project 3"})

	err := repo.DeleteByNamespaceCode(ctx, "ns-to-delete")
	assert.NoError(t, err)

	projects, err := repo.FindByNamespace(ctx, "ns-to-delete")
	assert.NoError(t, err)
	assert.Empty(t, projects)

	projectsKept, err := repo.FindByNamespace(ctx, "ns-to-keep")
	assert.NoError(t, err)
	assert.Len(t, projectsKept, 1)
}

func TestProjectRepository_FindByCode(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(repo ProjectRepository, ctx context.Context)
		namespaceCode string
		projectCode   string
		wantName      string
		wantErr       bool
	}{
		{
			name: "find existing project",
			setupFunc: func(repo ProjectRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.Project{
					ProjectCode:   "find-me",
					NamespaceCode: "test-ns",
					Name:          "Find Me",
				})
			},
			namespaceCode: "test-ns",
			projectCode:   "find-me",
			wantName:      "Find Me",
			wantErr:       false,
		},
		{
			name:          "find non-existing project",
			setupFunc:     func(repo ProjectRepository, ctx context.Context) {},
			namespaceCode: "test-ns",
			projectCode:   "not-found",
			wantErr:       true,
		},
		{
			name: "find project wrong namespace",
			setupFunc: func(repo ProjectRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.Project{
					ProjectCode:   "wrong-ns",
					NamespaceCode: "test-ns",
					Name:          "Wrong NS",
				})
			},
			namespaceCode: "other-ns",
			projectCode:   "wrong-ns",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupProjectTestDB(t)
			createTestNamespace(t, db, "test-ns", "Test Namespace")
			createTestNamespace(t, db, "other-ns", "Other Namespace")
			repo := NewProjectRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			result, err := repo.FindByCode(ctx, tt.namespaceCode, tt.projectCode)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantName, result.Name)
				assert.Equal(t, tt.projectCode, result.ProjectCode)
				assert.Equal(t, tt.namespaceCode, result.NamespaceCode)
			}
		})
	}
}

func TestProjectRepository_FindByCodeWithNamespace(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Project{
		ProjectCode:   "with-ns",
		NamespaceCode: "test-ns",
		Name:          "With Namespace",
	})

	result, err := repo.FindByCodeWithNamespace(ctx, "test-ns", "with-ns")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "With Namespace", result.Name)
	assert.NotNil(t, result.Namespace)
	assert.Equal(t, "Test Namespace", result.Namespace.Name)
}

func TestProjectRepository_FindByCodeWithNamespace_NotFound(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	result, err := repo.FindByCodeWithNamespace(ctx, "test-ns", "not-found")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestProjectRepository_FindAll(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo ProjectRepository, ctx context.Context)
		wantCount int
	}{
		{
			name:      "find all with empty database",
			setupFunc: func(repo ProjectRepository, ctx context.Context) {},
			wantCount: 0,
		},
		{
			name: "find all with multiple projects",
			setupFunc: func(repo ProjectRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-1", NamespaceCode: "test-ns", Name: "Project 1"})
				_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-2", NamespaceCode: "test-ns", Name: "Project 2"})
				_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-3", NamespaceCode: "test-ns", Name: "Project 3"})
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupProjectTestDB(t)
			createTestNamespace(t, db, "test-ns", "Test Namespace")
			repo := NewProjectRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			result, err := repo.FindAll(ctx)

			assert.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestProjectRepository_FindByNamespace(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "ns-1", "Namespace 1")
	createTestNamespace(t, db, "ns-2", "Namespace 2")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-1", NamespaceCode: "ns-1", Name: "Project 1"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-2", NamespaceCode: "ns-1", Name: "Project 2"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-3", NamespaceCode: "ns-2", Name: "Project 3"})

	t.Run("find projects in ns-1", func(t *testing.T) {
		results, err := repo.FindByNamespace(ctx, "ns-1")
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("find projects in ns-2", func(t *testing.T) {
		results, err := repo.FindByNamespace(ctx, "ns-2")
		assert.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("find projects in non-existing namespace", func(t *testing.T) {
		results, err := repo.FindByNamespace(ctx, "ns-3")
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestProjectRepository_Search(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Project{ProjectCode: "search-1", NamespaceCode: "test-ns", Name: "Alpha"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "search-2", NamespaceCode: "test-ns", Name: "Beta"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "search-3", NamespaceCode: "test-ns", Name: "Gamma"})

	t.Run("search with nil query returns all", func(t *testing.T) {
		results, err := repo.Search(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("search with custom query", func(t *testing.T) {
		query := db.Model(&model.Project{}).Where("name = ?", "Alpha")
		results, err := repo.Search(ctx, query)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Alpha", results[0].Name)
	})
}

func TestProjectRepository_SearchPaginate(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		_ = repo.Create(ctx, &model.Project{
			ProjectCode:   "paginate-" + string(rune('a'+i-1)),
			NamespaceCode: "test-ns",
			Name:          "Project " + string(rune('A'+i-1)),
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
			wantTotal: 10,
		},
		{
			name:      "paginate with offset",
			query:     nil,
			limit:     5,
			offset:    5,
			wantCount: 5,
			wantTotal: 10,
		},
		{
			name:      "paginate with offset beyond total",
			query:     nil,
			limit:     5,
			offset:    15,
			wantCount: 0,
			wantTotal: 10,
		},
		{
			name:      "paginate without limit returns all",
			query:     nil,
			limit:     0,
			offset:    0,
			wantCount: 10,
			wantTotal: 10,
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

func TestProjectRepository_SearchPaginate_PreloadsNamespace(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Project{ProjectCode: "preload-test", NamespaceCode: "test-ns", Name: "Preload Test"})

	results, _, err := repo.SearchPaginate(ctx, nil, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NotNil(t, results[0].Namespace)
	assert.Equal(t, "Test Namespace", results[0].Namespace.Name)
}

func TestProjectRepository_CountRedirects(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-1", NamespaceCode: "test-ns", Name: "Project 1"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-2", NamespaceCode: "test-ns", Name: "Project 2"})

	isPublished := true
	_ = db.Create(&model.Redirect{NamespaceCode: "test-ns", ProjectCode: "proj-1", IsPublished: &isPublished}).Error
	_ = db.Create(&model.Redirect{NamespaceCode: "test-ns", ProjectCode: "proj-1", IsPublished: &isPublished}).Error
	_ = db.Create(&model.Redirect{NamespaceCode: "test-ns", ProjectCode: "proj-2", IsPublished: &isPublished}).Error

	t.Run("count redirects for proj-1", func(t *testing.T) {
		count, err := repo.CountRedirects(ctx, "test-ns", "proj-1")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("count redirects for proj-2", func(t *testing.T) {
		count, err := repo.CountRedirects(ctx, "test-ns", "proj-2")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("count redirects for non-existing project", func(t *testing.T) {
		count, err := repo.CountRedirects(ctx, "test-ns", "non-existing")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestProjectRepository_CountRedirectDrafts(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-1", NamespaceCode: "test-ns", Name: "Project 1"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-2", NamespaceCode: "test-ns", Name: "Project 2"})

	_ = db.Create(&model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "proj-1", ChangeType: model.DraftChangeTypeCreate}).Error
	_ = db.Create(&model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "proj-1", ChangeType: model.DraftChangeTypeUpdate}).Error
	_ = db.Create(&model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "proj-1", ChangeType: model.DraftChangeTypeDelete}).Error
	_ = db.Create(&model.RedirectDraft{NamespaceCode: "test-ns", ProjectCode: "proj-2", ChangeType: model.DraftChangeTypeCreate}).Error

	t.Run("count redirect drafts for proj-1", func(t *testing.T) {
		count, err := repo.CountRedirectDrafts(ctx, "test-ns", "proj-1")
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("count redirect drafts for proj-2", func(t *testing.T) {
		count, err := repo.CountRedirectDrafts(ctx, "test-ns", "proj-2")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("count redirect drafts for non-existing project", func(t *testing.T) {
		count, err := repo.CountRedirectDrafts(ctx, "test-ns", "non-existing")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestProjectRepository_CountPages(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-1", NamespaceCode: "test-ns", Name: "Project 1"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-2", NamespaceCode: "test-ns", Name: "Project 2"})

	isPublished := true
	_ = db.Create(&model.Page{NamespaceCode: "test-ns", ProjectCode: "proj-1", IsPublished: &isPublished}).Error
	_ = db.Create(&model.Page{NamespaceCode: "test-ns", ProjectCode: "proj-1", IsPublished: &isPublished}).Error
	_ = db.Create(&model.Page{NamespaceCode: "test-ns", ProjectCode: "proj-2", IsPublished: &isPublished}).Error

	t.Run("count pages for proj-1", func(t *testing.T) {
		count, err := repo.CountPages(ctx, "test-ns", "proj-1")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("count pages for proj-2", func(t *testing.T) {
		count, err := repo.CountPages(ctx, "test-ns", "proj-2")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("count pages for non-existing project", func(t *testing.T) {
		count, err := repo.CountPages(ctx, "test-ns", "non-existing")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestProjectRepository_CountPageDrafts(t *testing.T) {
	db := setupProjectTestDB(t)
	createTestNamespace(t, db, "test-ns", "Test Namespace")
	repo := NewProjectRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-1", NamespaceCode: "test-ns", Name: "Project 1"})
	_ = repo.Create(ctx, &model.Project{ProjectCode: "proj-2", NamespaceCode: "test-ns", Name: "Project 2"})

	_ = db.Create(&model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "proj-1", ChangeType: model.DraftChangeTypeCreate}).Error
	_ = db.Create(&model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "proj-1", ChangeType: model.DraftChangeTypeUpdate}).Error
	_ = db.Create(&model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "proj-1", ChangeType: model.DraftChangeTypeDelete}).Error
	_ = db.Create(&model.PageDraft{NamespaceCode: "test-ns", ProjectCode: "proj-2", ChangeType: model.DraftChangeTypeCreate}).Error

	t.Run("count page drafts for proj-1", func(t *testing.T) {
		count, err := repo.CountPageDrafts(ctx, "test-ns", "proj-1")
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("count page drafts for proj-2", func(t *testing.T) {
		count, err := repo.CountPageDrafts(ctx, "test-ns", "proj-2")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("count page drafts for non-existing project", func(t *testing.T) {
		count, err := repo.CountPageDrafts(ctx, "test-ns", "non-existing")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}
