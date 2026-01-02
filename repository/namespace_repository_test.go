package repository

import (
	"context"
	"testing"

	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupNamespaceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Namespace{})
	assert.NoError(t, err)

	return db
}

func TestNewNamespaceRepository(t *testing.T) {
	db := setupNamespaceTestDB(t)
	repo := NewNamespaceRepository(db)

	assert.NotNil(t, repo)
}

func TestNamespaceRepository_GetTx(t *testing.T) {
	db := setupNamespaceTestDB(t)
	repo := NewNamespaceRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var namespaces []model.Namespace
	err := tx.Find(&namespaces).Error
	assert.NoError(t, err)
}

func TestNamespaceRepository_GetQuery(t *testing.T) {
	db := setupNamespaceTestDB(t)
	repo := NewNamespaceRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	// Verify it's a valid query by executing it
	var namespaces []model.Namespace
	err := query.Find(&namespaces).Error
	assert.NoError(t, err)
}

func TestNamespaceRepository_Create(t *testing.T) {
	tests := []struct {
		name      string
		namespace *model.Namespace
		wantErr   bool
	}{
		{
			name: "create valid namespace",
			namespace: &model.Namespace{
				NamespaceCode: "test-ns",
				Name:          "Test Namespace",
			},
			wantErr: false,
		},
		{
			name: "create namespace with empty code",
			namespace: &model.Namespace{
				NamespaceCode: "",
				Name:          "Empty Code Namespace",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupNamespaceTestDB(t)
			repo := NewNamespaceRepository(db)
			ctx := context.Background()

			err := repo.Create(ctx, tt.namespace)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.namespace.ID)
			}
		})
	}
}

func TestNamespaceRepository_Create_DuplicateCode(t *testing.T) {
	db := setupNamespaceTestDB(t)
	repo := NewNamespaceRepository(db)
	ctx := context.Background()

	ns1 := &model.Namespace{
		NamespaceCode: "duplicate-code",
		Name:          "First Namespace",
	}
	err := repo.Create(ctx, ns1)
	assert.NoError(t, err)

	ns2 := &model.Namespace{
		NamespaceCode: "duplicate-code",
		Name:          "Second Namespace",
	}
	err = repo.Create(ctx, ns2)
	assert.Error(t, err)
}

func TestNamespaceRepository_Update(t *testing.T) {
	db := setupNamespaceTestDB(t)
	repo := NewNamespaceRepository(db)
	ctx := context.Background()

	ns := &model.Namespace{
		NamespaceCode: "update-ns",
		Name:          "Original Name",
	}
	err := repo.Create(ctx, ns)
	assert.NoError(t, err)

	ns.Name = "Updated Name"
	err = repo.Update(ctx, ns)
	assert.NoError(t, err)

	found, err := repo.FindByCode(ctx, "update-ns")
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name)
}

func TestNamespaceRepository_DeleteByCode(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(repo NamespaceRepository, ctx context.Context)
		deleteCode string
		wantErr    bool
	}{
		{
			name: "delete existing namespace",
			setupFunc: func(repo NamespaceRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.Namespace{
					NamespaceCode: "to-delete",
					Name:          "To Delete",
				})
			},
			deleteCode: "to-delete",
			wantErr:    false,
		},
		{
			name:       "delete non-existing namespace",
			setupFunc:  func(repo NamespaceRepository, ctx context.Context) {},
			deleteCode: "non-existing",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupNamespaceTestDB(t)
			repo := NewNamespaceRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			err := repo.DeleteByCode(ctx, tt.deleteCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNamespaceRepository_FindByCode(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo NamespaceRepository, ctx context.Context)
		code      string
		wantName  string
		wantErr   bool
	}{
		{
			name: "find existing namespace",
			setupFunc: func(repo NamespaceRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.Namespace{
					NamespaceCode: "find-me",
					Name:          "Find Me",
				})
			},
			code:     "find-me",
			wantName: "Find Me",
			wantErr:  false,
		},
		{
			name:      "find non-existing namespace",
			setupFunc: func(repo NamespaceRepository, ctx context.Context) {},
			code:      "not-found",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupNamespaceTestDB(t)
			repo := NewNamespaceRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			result, err := repo.FindByCode(ctx, tt.code)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantName, result.Name)
				assert.Equal(t, tt.code, result.NamespaceCode)
			}
		})
	}
}

func TestNamespaceRepository_FindAll(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo NamespaceRepository, ctx context.Context)
		wantCount int
	}{
		{
			name:      "find all with empty database",
			setupFunc: func(repo NamespaceRepository, ctx context.Context) {},
			wantCount: 0,
		},
		{
			name: "find all with multiple namespaces",
			setupFunc: func(repo NamespaceRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.Namespace{NamespaceCode: "ns-1", Name: "Namespace 1"})
				_ = repo.Create(ctx, &model.Namespace{NamespaceCode: "ns-2", Name: "Namespace 2"})
				_ = repo.Create(ctx, &model.Namespace{NamespaceCode: "ns-3", Name: "Namespace 3"})
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupNamespaceTestDB(t)
			repo := NewNamespaceRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			result, err := repo.FindAll(ctx)

			assert.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestNamespaceRepository_Search(t *testing.T) {
	db := setupNamespaceTestDB(t)
	repo := NewNamespaceRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Namespace{NamespaceCode: "search-1", Name: "Alpha"})
	_ = repo.Create(ctx, &model.Namespace{NamespaceCode: "search-2", Name: "Beta"})
	_ = repo.Create(ctx, &model.Namespace{NamespaceCode: "search-3", Name: "Gamma"})

	t.Run("search with nil query returns all", func(t *testing.T) {
		results, err := repo.Search(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("search with custom query", func(t *testing.T) {
		query := db.Model(&model.Namespace{}).Where("name = ?", "Alpha")
		results, err := repo.Search(ctx, query)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Alpha", results[0].Name)
	})
}

func TestNamespaceRepository_SearchPaginate(t *testing.T) {
	db := setupNamespaceTestDB(t)
	repo := NewNamespaceRepository(db)
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		_ = repo.Create(ctx, &model.Namespace{
			NamespaceCode: "paginate-" + string(rune('a'+i-1)),
			Name:          "Namespace " + string(rune('A'+i-1)),
		})
	}

	baseQuery := func() *gorm.DB {
		return db.Model(&model.Namespace{})
	}

	t.Run("paginate with limit", func(t *testing.T) {
		results, total, err := repo.SearchPaginate(ctx, baseQuery(), 5, 0)
		assert.NoError(t, err)
		assert.Len(t, results, 5)
		assert.Equal(t, int64(10), total)
	})

	t.Run("paginate with offset", func(t *testing.T) {
		results, total, err := repo.SearchPaginate(ctx, baseQuery(), 5, 5)
		assert.NoError(t, err)
		assert.Len(t, results, 5)
		assert.Equal(t, int64(10), total)
	})

	t.Run("paginate with offset beyond total", func(t *testing.T) {
		results, total, err := repo.SearchPaginate(ctx, baseQuery(), 5, 15)
		assert.NoError(t, err)
		assert.Len(t, results, 0)
		assert.Equal(t, int64(10), total)
	})

	t.Run("paginate without limit returns all", func(t *testing.T) {
		results, total, err := repo.SearchPaginate(ctx, baseQuery(), 0, 0)
		assert.NoError(t, err)
		assert.Len(t, results, 10)
		assert.Equal(t, int64(10), total)
	})
}

func TestNamespaceRepository_SearchPaginate_WithCustomQuery(t *testing.T) {
	db := setupNamespaceTestDB(t)
	repo := NewNamespaceRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Namespace{NamespaceCode: "alpha-1", Name: "Alpha One"})
	_ = repo.Create(ctx, &model.Namespace{NamespaceCode: "alpha-2", Name: "Alpha Two"})
	_ = repo.Create(ctx, &model.Namespace{NamespaceCode: "beta-1", Name: "Beta One"})

	query := db.Model(&model.Namespace{}).Where("namespace_code LIKE ?", "alpha%")
	results, total, err := repo.SearchPaginate(ctx, query, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, int64(2), total)
}
