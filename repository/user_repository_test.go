package repository

import (
	"context"
	"testing"

	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func boolPtr(b bool) *bool {
	return &b
}

func setupUserTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.User{})
	assert.NoError(t, err)

	return db
}

func TestNewUserRepository(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)

	assert.NotNil(t, repo)
}

func TestUserRepository_GetTx(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var users []model.User
	err := tx.Find(&users).Error
	assert.NoError(t, err)
}

func TestUserRepository_GetQuery(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var users []model.User
	err := query.Find(&users).Error
	assert.NoError(t, err)
}

func TestUserRepository_Create(t *testing.T) {
	tests := []struct {
		name    string
		user    *model.User
		wantErr bool
	}{
		{
			name: "create valid user",
			user: &model.User{
				Username:  "testuser",
				Password:  "hashedpassword",
				Firstname: "Test",
				Lastname:  "User",
				Active:    boolPtr(true),
			},
			wantErr: false,
		},
		{
			name: "create user without password",
			user: &model.User{
				Username:  "nopassuser",
				Firstname: "No",
				Lastname:  "Password",
				Active:    boolPtr(true),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserTestDB(t)
			repo := NewUserRepository(db)
			ctx := context.Background()

			err := repo.Create(ctx, tt.user)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.user.ID)
			}
		})
	}
}

func TestUserRepository_Create_DuplicateUsername(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user1 := &model.User{
		Username: "duplicate",
		Active:   boolPtr(true),
	}
	err := repo.Create(ctx, user1)
	assert.NoError(t, err)

	user2 := &model.User{
		Username: "duplicate",
		Active:   boolPtr(true),
	}
	err = repo.Create(ctx, user2)
	assert.Error(t, err)
}

func TestUserRepository_Update(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:  "updateuser",
		Firstname: "Original",
		Lastname:  "Name",
		Active:    boolPtr(true),
	}
	err := repo.Create(ctx, user)
	assert.NoError(t, err)

	user.Firstname = "Updated"
	user.Lastname = "Person"
	err = repo.Update(ctx, user)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated", found.Firstname)
	assert.Equal(t, "Person", found.Lastname)
}

func TestUserRepository_Delete(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo UserRepository, ctx context.Context) int64
		wantErr   bool
	}{
		{
			name: "delete existing user",
			setupFunc: func(repo UserRepository, ctx context.Context) int64 {
				user := &model.User{Username: "todelete", Active: boolPtr(true)}
				_ = repo.Create(ctx, user)
				return user.ID
			},
			wantErr: false,
		},
		{
			name: "delete non-existing user",
			setupFunc: func(repo UserRepository, ctx context.Context) int64 {
				return 9999
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserTestDB(t)
			repo := NewUserRepository(db)
			ctx := context.Background()

			id := tt.setupFunc(repo, ctx)

			err := repo.Delete(ctx, id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo UserRepository, ctx context.Context) int64
		wantErr   bool
	}{
		{
			name: "find existing user",
			setupFunc: func(repo UserRepository, ctx context.Context) int64 {
				user := &model.User{Username: "findme", Firstname: "Find", Active: boolPtr(true)}
				_ = repo.Create(ctx, user)
				return user.ID
			},
			wantErr: false,
		},
		{
			name: "find non-existing user",
			setupFunc: func(repo UserRepository, ctx context.Context) int64 {
				return 9999
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserTestDB(t)
			repo := NewUserRepository(db)
			ctx := context.Background()

			id := tt.setupFunc(repo, ctx)

			result, err := repo.FindByID(ctx, id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, id, result.ID)
			}
		})
	}
}

func TestUserRepository_FindByUsername(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo UserRepository, ctx context.Context)
		username  string
		wantErr   bool
	}{
		{
			name: "find existing user",
			setupFunc: func(repo UserRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.User{Username: "findbyname", Active: boolPtr(true)})
			},
			username: "findbyname",
			wantErr:  false,
		},
		{
			name:      "find non-existing user",
			setupFunc: func(repo UserRepository, ctx context.Context) {},
			username:  "notfound",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserTestDB(t)
			repo := NewUserRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			result, err := repo.FindByUsername(ctx, tt.username)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.username, result.Username)
			}
		})
	}
}

func TestUserRepository_FindAll(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo UserRepository, ctx context.Context)
		wantCount int
	}{
		{
			name:      "find all with empty database",
			setupFunc: func(repo UserRepository, ctx context.Context) {},
			wantCount: 0,
		},
		{
			name: "find all with multiple users",
			setupFunc: func(repo UserRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.User{Username: "user1", Active: boolPtr(true)})
				_ = repo.Create(ctx, &model.User{Username: "user2", Active: boolPtr(true)})
				_ = repo.Create(ctx, &model.User{Username: "user3", Active: boolPtr(false)})
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserTestDB(t)
			repo := NewUserRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			result, err := repo.FindAll(ctx)

			assert.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestUserRepository_Search(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.User{Username: "alice", Firstname: "Alice", Active: boolPtr(true)})
	_ = repo.Create(ctx, &model.User{Username: "bob", Firstname: "Bob", Active: boolPtr(true)})
	_ = repo.Create(ctx, &model.User{Username: "charlie", Firstname: "Charlie", Active: boolPtr(false)})

	t.Run("search with nil query returns all", func(t *testing.T) {
		results, err := repo.Search(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("search with custom query", func(t *testing.T) {
		query := db.Model(&model.User{}).Where("active = ?", true)
		results, err := repo.Search(ctx, query)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestUserRepository_SearchPaginate(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		_ = repo.Create(ctx, &model.User{
			Username: "user" + string(rune('a'+i-1)),
			Active:   boolPtr(true),
		})
	}

	baseQuery := func() *gorm.DB {
		return db.Model(&model.User{})
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

func TestUserRepository_UpdatePassword(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username: "passuser",
		Password: "oldhash",
		Active:   boolPtr(true),
	}
	err := repo.Create(ctx, user)
	assert.NoError(t, err)

	err = repo.UpdatePassword(ctx, user.ID, "newhash")
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "newhash", found.Password)
}

func TestUserRepository_UpdateStatus(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username: "statususer",
		Active:   boolPtr(true),
	}
	err := repo.Create(ctx, user)
	assert.NoError(t, err)

	t.Run("deactivate user", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, user.ID, false)
		assert.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.False(t, *found.Active)
	})

	t.Run("activate user", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, user.ID, true)
		assert.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.True(t, *found.Active)
	})
}

func TestUserRepository_UpdateRefreshTokenHash(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:         "tokenuser",
		RefreshTokenHash: "oldhash",
		Active:           boolPtr(true),
	}
	err := repo.Create(ctx, user)
	assert.NoError(t, err)

	t.Run("update refresh token hash", func(t *testing.T) {
		err := repo.UpdateRefreshTokenHash(ctx, user.ID, "newtokenhash")
		assert.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, "newtokenhash", found.RefreshTokenHash)
	})

	t.Run("clear refresh token hash", func(t *testing.T) {
		err := repo.UpdateRefreshTokenHash(ctx, user.ID, "")
		assert.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, "", found.RefreshTokenHash)
	})
}
