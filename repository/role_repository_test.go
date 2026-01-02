package repository

import (
	"context"
	"testing"

	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRoleTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.User{}, &model.Role{}, &model.UserRole{}, &model.AdminPermission{}, &model.ResourcePermission{})
	assert.NoError(t, err)

	return db
}

func TestNewRoleRepository(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)

	assert.NotNil(t, repo)
}

func TestRoleRepository_GetTx(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var roles []model.Role
	err := tx.Find(&roles).Error
	assert.NoError(t, err)
}

func TestRoleRepository_GetQuery(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var roles []model.Role
	err := query.Find(&roles).Error
	assert.NoError(t, err)
}

func TestRoleRepository_Create(t *testing.T) {
	tests := []struct {
		name    string
		role    *model.Role
		wantErr bool
	}{
		{
			name: "create valid role",
			role: &model.Role{
				Code: "admin",
				Type: model.RoleTypeRole,
			},
			wantErr: false,
		},
		{
			name: "create user type role",
			role: &model.Role{
				Code: "user.1",
				Type: model.RoleTypeUser,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupRoleTestDB(t)
			repo := NewRoleRepository(db)
			ctx := context.Background()

			err := repo.Create(ctx, tt.role)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.role.ID)
			}
		})
	}
}

func TestRoleRepository_Create_DuplicateCode(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role1 := &model.Role{Code: "duplicate", Type: model.RoleTypeRole}
	err := repo.Create(ctx, role1)
	assert.NoError(t, err)

	role2 := &model.Role{Code: "duplicate", Type: model.RoleTypeRole}
	err = repo.Create(ctx, role2)
	assert.Error(t, err)
}

func TestRoleRepository_Update(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	role := &model.Role{Code: "original", Type: model.RoleTypeRole}
	err := repo.Create(ctx, role)
	assert.NoError(t, err)

	role.Code = "updated"
	err = repo.Update(ctx, role)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, role.ID)
	assert.NoError(t, err)
	assert.Equal(t, "updated", found.Code)
}

func TestRoleRepository_Delete(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create user and role
	user := &model.User{Username: "testuser", Active: boolPtr(true)}
	err := userRepo.Create(ctx, user)
	assert.NoError(t, err)

	role := &model.Role{Code: "todelete", Type: model.RoleTypeRole}
	err = repo.Create(ctx, role)
	assert.NoError(t, err)

	// Add user to role
	err = repo.AddUserToRole(ctx, user.ID, role.ID)
	assert.NoError(t, err)

	// Delete role (should also delete user_roles)
	err = repo.Delete(ctx, role.ID)
	assert.NoError(t, err)

	// Verify role is deleted
	_, err = repo.FindByID(ctx, role.ID)
	assert.Error(t, err)

	// Verify user_role is deleted
	hasRole, err := repo.HasUserRole(ctx, user.ID, role.ID)
	assert.NoError(t, err)
	assert.False(t, hasRole)
}

func TestRoleRepository_FindByID(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo RoleRepository, ctx context.Context) int64
		wantErr   bool
	}{
		{
			name: "find existing role",
			setupFunc: func(repo RoleRepository, ctx context.Context) int64 {
				role := &model.Role{Code: "findme", Type: model.RoleTypeRole}
				_ = repo.Create(ctx, role)
				return role.ID
			},
			wantErr: false,
		},
		{
			name: "find non-existing role",
			setupFunc: func(repo RoleRepository, ctx context.Context) int64 {
				return 9999
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupRoleTestDB(t)
			repo := NewRoleRepository(db)
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

func TestRoleRepository_FindByCode(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo RoleRepository, ctx context.Context)
		code      string
		wantErr   bool
	}{
		{
			name: "find existing role",
			setupFunc: func(repo RoleRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.Role{Code: "findbycode", Type: model.RoleTypeRole})
			},
			code:    "findbycode",
			wantErr: false,
		},
		{
			name:      "find non-existing role",
			setupFunc: func(repo RoleRepository, ctx context.Context) {},
			code:      "notfound",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupRoleTestDB(t)
			repo := NewRoleRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			result, err := repo.FindByCode(ctx, tt.code)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.code, result.Code)
			}
		})
	}
}

func TestRoleRepository_FindByCodeAndType(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Role{Code: "admin", Type: model.RoleTypeRole})
	_ = repo.Create(ctx, &model.Role{Code: "user.1", Type: model.RoleTypeUser})

	t.Run("find role type", func(t *testing.T) {
		result, err := repo.FindByCodeAndType(ctx, "admin", model.RoleTypeRole)
		assert.NoError(t, err)
		assert.Equal(t, "admin", result.Code)
		assert.Equal(t, model.RoleTypeRole, result.Type)
	})

	t.Run("find user type", func(t *testing.T) {
		result, err := repo.FindByCodeAndType(ctx, "user.1", model.RoleTypeUser)
		assert.NoError(t, err)
		assert.Equal(t, "user.1", result.Code)
		assert.Equal(t, model.RoleTypeUser, result.Type)
	})

	t.Run("wrong type returns error", func(t *testing.T) {
		_, err := repo.FindByCodeAndType(ctx, "admin", model.RoleTypeUser)
		assert.Error(t, err)
	})
}

func TestRoleRepository_FindAll(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(repo RoleRepository, ctx context.Context)
		wantCount int
	}{
		{
			name:      "find all with empty database",
			setupFunc: func(repo RoleRepository, ctx context.Context) {},
			wantCount: 0,
		},
		{
			name: "find all with multiple roles",
			setupFunc: func(repo RoleRepository, ctx context.Context) {
				_ = repo.Create(ctx, &model.Role{Code: "role1", Type: model.RoleTypeRole})
				_ = repo.Create(ctx, &model.Role{Code: "role2", Type: model.RoleTypeRole})
				_ = repo.Create(ctx, &model.Role{Code: "user.1", Type: model.RoleTypeUser})
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupRoleTestDB(t)
			repo := NewRoleRepository(db)
			ctx := context.Background()

			tt.setupFunc(repo, ctx)

			result, err := repo.FindAll(ctx)

			assert.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestRoleRepository_FindAllByType(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	_ = repo.Create(ctx, &model.Role{Code: "admin", Type: model.RoleTypeRole})
	_ = repo.Create(ctx, &model.Role{Code: "editor", Type: model.RoleTypeRole})
	_ = repo.Create(ctx, &model.Role{Code: "user.1", Type: model.RoleTypeUser})
	_ = repo.Create(ctx, &model.Role{Code: "user.2", Type: model.RoleTypeUser})

	t.Run("find all role type", func(t *testing.T) {
		result, err := repo.FindAllByType(ctx, model.RoleTypeRole)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("find all user type", func(t *testing.T) {
		result, err := repo.FindAllByType(ctx, model.RoleTypeUser)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestRoleRepository_SearchPaginate(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		_ = repo.Create(ctx, &model.Role{
			Code: "role" + string(rune('a'+i-1)),
			Type: model.RoleTypeRole,
		})
	}

	t.Run("paginate with limit", func(t *testing.T) {
		results, total, err := repo.SearchPaginate(ctx, nil, 5, 0)
		assert.NoError(t, err)
		assert.Len(t, results, 5)
		assert.Equal(t, int64(10), total)
	})

	t.Run("paginate with offset", func(t *testing.T) {
		results, total, err := repo.SearchPaginate(ctx, nil, 5, 5)
		assert.NoError(t, err)
		assert.Len(t, results, 5)
		assert.Equal(t, int64(10), total)
	})

	t.Run("paginate without limit returns all", func(t *testing.T) {
		results, total, err := repo.SearchPaginate(ctx, nil, 0, 0)
		assert.NoError(t, err)
		assert.Len(t, results, 10)
		assert.Equal(t, int64(10), total)
	})
}

func TestRoleRepository_AddUserToRole(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", Active: boolPtr(true)}
	err := userRepo.Create(ctx, user)
	assert.NoError(t, err)

	role := &model.Role{Code: "testrole", Type: model.RoleTypeRole}
	err = repo.Create(ctx, role)
	assert.NoError(t, err)

	t.Run("add user to role", func(t *testing.T) {
		err := repo.AddUserToRole(ctx, user.ID, role.ID)
		assert.NoError(t, err)

		hasRole, err := repo.HasUserRole(ctx, user.ID, role.ID)
		assert.NoError(t, err)
		assert.True(t, hasRole)
	})

	t.Run("add duplicate fails", func(t *testing.T) {
		err := repo.AddUserToRole(ctx, user.ID, role.ID)
		assert.Error(t, err)
	})
}

func TestRoleRepository_RemoveUserFromRole(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", Active: boolPtr(true)}
	err := userRepo.Create(ctx, user)
	assert.NoError(t, err)

	role := &model.Role{Code: "testrole", Type: model.RoleTypeRole}
	err = repo.Create(ctx, role)
	assert.NoError(t, err)

	err = repo.AddUserToRole(ctx, user.ID, role.ID)
	assert.NoError(t, err)

	t.Run("remove user from role", func(t *testing.T) {
		err := repo.RemoveUserFromRole(ctx, user.ID, role.ID)
		assert.NoError(t, err)

		hasRole, err := repo.HasUserRole(ctx, user.ID, role.ID)
		assert.NoError(t, err)
		assert.False(t, hasRole)
	})
}

func TestRoleRepository_GetUserRoles(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", Active: boolPtr(true)}
	err := userRepo.Create(ctx, user)
	assert.NoError(t, err)

	role1 := &model.Role{Code: "role1", Type: model.RoleTypeRole}
	role2 := &model.Role{Code: "role2", Type: model.RoleTypeRole}
	_ = repo.Create(ctx, role1)
	_ = repo.Create(ctx, role2)

	_ = repo.AddUserToRole(ctx, user.ID, role1.ID)
	_ = repo.AddUserToRole(ctx, user.ID, role2.ID)

	t.Run("get user roles", func(t *testing.T) {
		roles, err := repo.GetUserRoles(ctx, user.ID)
		assert.NoError(t, err)
		assert.Len(t, roles, 2)
	})

	t.Run("get roles for user without roles", func(t *testing.T) {
		user2 := &model.User{Username: "noroles", Active: boolPtr(true)}
		_ = userRepo.Create(ctx, user2)

		roles, err := repo.GetUserRoles(ctx, user2.ID)
		assert.NoError(t, err)
		assert.Len(t, roles, 0)
	})
}

func TestRoleRepository_GetRoleUsers(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	role := &model.Role{Code: "testrole", Type: model.RoleTypeRole}
	err := repo.Create(ctx, role)
	assert.NoError(t, err)

	user1 := &model.User{Username: "user1", Active: boolPtr(true)}
	user2 := &model.User{Username: "user2", Active: boolPtr(true)}
	_ = userRepo.Create(ctx, user1)
	_ = userRepo.Create(ctx, user2)

	_ = repo.AddUserToRole(ctx, user1.ID, role.ID)
	_ = repo.AddUserToRole(ctx, user2.ID, role.ID)

	t.Run("get role users", func(t *testing.T) {
		users, err := repo.GetRoleUsers(ctx, role.ID)
		assert.NoError(t, err)
		assert.Len(t, users, 2)
	})

	t.Run("get users for role without users", func(t *testing.T) {
		emptyRole := &model.Role{Code: "emptyrole", Type: model.RoleTypeRole}
		_ = repo.Create(ctx, emptyRole)

		users, err := repo.GetRoleUsers(ctx, emptyRole.ID)
		assert.NoError(t, err)
		assert.Len(t, users, 0)
	})
}

func TestRoleRepository_HasUserRole(t *testing.T) {
	db := setupRoleTestDB(t)
	repo := NewRoleRepository(db)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", Active: boolPtr(true)}
	_ = userRepo.Create(ctx, user)

	role := &model.Role{Code: "testrole", Type: model.RoleTypeRole}
	_ = repo.Create(ctx, role)

	t.Run("user does not have role", func(t *testing.T) {
		hasRole, err := repo.HasUserRole(ctx, user.ID, role.ID)
		assert.NoError(t, err)
		assert.False(t, hasRole)
	})

	_ = repo.AddUserToRole(ctx, user.ID, role.ID)

	t.Run("user has role", func(t *testing.T) {
		hasRole, err := repo.HasUserRole(ctx, user.ID, role.ID)
		assert.NoError(t, err)
		assert.True(t, hasRole)
	})
}
