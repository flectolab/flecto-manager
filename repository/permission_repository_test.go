package repository

import (
	"context"
	"testing"

	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPermissionTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Role{}, &model.ResourcePermission{}, &model.AdminPermission{})
	assert.NoError(t, err)

	return db
}

func createTestRole(t *testing.T, db *gorm.DB, code string) *model.Role {
	role := &model.Role{Code: code, Type: model.RoleTypeRole}
	err := db.Create(role).Error
	assert.NoError(t, err)
	return role
}

// ResourcePermissionRepository Tests

func TestNewResourcePermissionRepository(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)

	assert.NotNil(t, repo)
}

func TestResourcePermissionRepository_GetTx(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var perms []model.ResourcePermission
	err := tx.Find(&perms).Error
	assert.NoError(t, err)
}

func TestResourcePermissionRepository_GetQuery(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var perms []model.ResourcePermission
	err := query.Find(&perms).Error
	assert.NoError(t, err)
}

func TestResourcePermissionRepository_Create(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	perm := &model.ResourcePermission{
		RoleID:    role.ID,
		Namespace: "prod",
		Project:   "api",
		Action:    model.ActionRead,
	}

	err := repo.Create(ctx, perm)

	assert.NoError(t, err)
	assert.NotZero(t, perm.ID)
}

func TestResourcePermissionRepository_Update(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	perm := &model.ResourcePermission{
		RoleID:    role.ID,
		Namespace: "prod",
		Project:   "api",
		Action:    model.ActionRead,
	}
	err := repo.Create(ctx, perm)
	assert.NoError(t, err)

	perm.Action = model.ActionWrite
	err = repo.Update(ctx, perm)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, perm.ID)
	assert.NoError(t, err)
	assert.Equal(t, model.ActionWrite, found.Action)
}

func TestResourcePermissionRepository_Delete(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	perm := &model.ResourcePermission{
		RoleID:    role.ID,
		Namespace: "prod",
		Action:    model.ActionRead,
	}
	err := repo.Create(ctx, perm)
	assert.NoError(t, err)

	err = repo.Delete(ctx, perm.ID)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, perm.ID)
	assert.Error(t, err)
}

func TestResourcePermissionRepository_FindByID(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	t.Run("find existing permission", func(t *testing.T) {
		perm := &model.ResourcePermission{
			RoleID:    role.ID,
			Namespace: "prod",
			Action:    model.ActionRead,
		}
		_ = repo.Create(ctx, perm)

		found, err := repo.FindByID(ctx, perm.ID)
		assert.NoError(t, err)
		assert.Equal(t, perm.ID, found.ID)
	})

	t.Run("find non-existing permission", func(t *testing.T) {
		_, err := repo.FindByID(ctx, 9999)
		assert.Error(t, err)
	})
}

func TestResourcePermissionRepository_FindByRoleID(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()
	role1 := createTestRole(t, db, "role1")
	role2 := createTestRole(t, db, "role2")

	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role1.ID, Namespace: "ns1", Action: model.ActionRead})
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role1.ID, Namespace: "ns2", Action: model.ActionWrite})
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role2.ID, Namespace: "ns1", Action: model.ActionRead})

	t.Run("find permissions for role1", func(t *testing.T) {
		perms, err := repo.FindByRoleID(ctx, role1.ID)
		assert.NoError(t, err)
		assert.Len(t, perms, 2)
	})

	t.Run("find permissions for role2", func(t *testing.T) {
		perms, err := repo.FindByRoleID(ctx, role2.ID)
		assert.NoError(t, err)
		assert.Len(t, perms, 1)
	})

	t.Run("find permissions for role without permissions", func(t *testing.T) {
		role3 := createTestRole(t, db, "role3")
		perms, err := repo.FindByRoleID(ctx, role3.ID)
		assert.NoError(t, err)
		assert.Len(t, perms, 0)
	})
}

func TestResourcePermissionRepository_FindByRoleIDs(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()
	role1 := createTestRole(t, db, "role1")
	role2 := createTestRole(t, db, "role2")
	role3 := createTestRole(t, db, "role3")

	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role1.ID, Namespace: "ns1", Action: model.ActionRead})
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role2.ID, Namespace: "ns2", Action: model.ActionWrite})
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role3.ID, Namespace: "ns3", Action: model.ActionWrite})

	t.Run("find permissions for multiple roles", func(t *testing.T) {
		perms, err := repo.FindByRoleIDs(ctx, []int64{role1.ID, role2.ID})
		assert.NoError(t, err)
		assert.Len(t, perms, 2)
	})

	t.Run("find permissions with empty role IDs", func(t *testing.T) {
		perms, err := repo.FindByRoleIDs(ctx, []int64{})
		assert.NoError(t, err)
		assert.Len(t, perms, 0)
	})
}

func TestResourcePermissionRepository_FindByNamespace(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role.ID, Namespace: "prod", Action: model.ActionRead})
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role.ID, Namespace: "prod", Action: model.ActionWrite})
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role.ID, Namespace: "staging", Action: model.ActionRead})

	t.Run("find by namespace", func(t *testing.T) {
		perms, err := repo.FindByNamespace(ctx, "prod")
		assert.NoError(t, err)
		assert.Len(t, perms, 2)
	})

	t.Run("find by non-existing namespace", func(t *testing.T) {
		perms, err := repo.FindByNamespace(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Len(t, perms, 0)
	})
}

func TestResourcePermissionRepository_FindByNamespaceAndProject(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	// Specific permissions
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role.ID, Namespace: "prod", Project: "api", Action: model.ActionRead})
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role.ID, Namespace: "prod", Project: "web", Action: model.ActionRead})
	// Wildcard namespace
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role.ID, Namespace: "*", Project: "api", Action: model.ActionWrite})
	// Wildcard project
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role.ID, Namespace: "prod", Project: "*", Action: model.ActionWrite})
	// Empty project (namespace level)
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role.ID, Namespace: "staging", Project: "", Action: model.ActionRead})

	t.Run("find exact match", func(t *testing.T) {
		perms, err := repo.FindByNamespaceAndProject(ctx, "prod", "api")
		assert.NoError(t, err)
		// Should match: exact, wildcard namespace, wildcard project
		assert.Len(t, perms, 3)
	})

	t.Run("find with wildcard namespace", func(t *testing.T) {
		perms, err := repo.FindByNamespaceAndProject(ctx, "dev", "api")
		assert.NoError(t, err)
		// Should match: wildcard namespace
		assert.Len(t, perms, 1)
	})

	t.Run("find namespace level permission", func(t *testing.T) {
		perms, err := repo.FindByNamespaceAndProject(ctx, "staging", "anyproject")
		assert.NoError(t, err)
		// Should match: empty project (namespace level)
		assert.Len(t, perms, 1)
	})
}

func TestResourcePermissionRepository_DeleteByRoleID(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewResourcePermissionRepository(db)
	ctx := context.Background()
	role1 := createTestRole(t, db, "role1")
	role2 := createTestRole(t, db, "role2")

	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role1.ID, Namespace: "ns1", Action: model.ActionRead})
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role1.ID, Namespace: "ns2", Action: model.ActionWrite})
	_ = repo.Create(ctx, &model.ResourcePermission{RoleID: role2.ID, Namespace: "ns1", Action: model.ActionRead})

	err := repo.DeleteByRoleID(ctx, role1.ID)
	assert.NoError(t, err)

	// role1 permissions should be deleted
	perms, err := repo.FindByRoleID(ctx, role1.ID)
	assert.NoError(t, err)
	assert.Len(t, perms, 0)

	// role2 permissions should remain
	perms, err = repo.FindByRoleID(ctx, role2.ID)
	assert.NoError(t, err)
	assert.Len(t, perms, 1)
}

// AdminPermissionRepository Tests

func TestNewAdminPermissionRepository(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)

	assert.NotNil(t, repo)
}

func TestAdminPermissionRepository_GetTx(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var perms []model.AdminPermission
	err := tx.Find(&perms).Error
	assert.NoError(t, err)
}

func TestAdminPermissionRepository_GetQuery(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var perms []model.AdminPermission
	err := query.Find(&perms).Error
	assert.NoError(t, err)
}

func TestAdminPermissionRepository_Create(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	perm := &model.AdminPermission{
		RoleID:  role.ID,
		Section: model.AdminSectionUsers,
		Action:  model.ActionRead,
	}

	err := repo.Create(ctx, perm)

	assert.NoError(t, err)
	assert.NotZero(t, perm.ID)
}

func TestAdminPermissionRepository_Update(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	perm := &model.AdminPermission{
		RoleID:  role.ID,
		Section: model.AdminSectionUsers,
		Action:  model.ActionRead,
	}
	err := repo.Create(ctx, perm)
	assert.NoError(t, err)

	perm.Action = model.ActionWrite
	err = repo.Update(ctx, perm)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, perm.ID)
	assert.NoError(t, err)
	assert.Equal(t, model.ActionWrite, found.Action)
}

func TestAdminPermissionRepository_Delete(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	perm := &model.AdminPermission{
		RoleID:  role.ID,
		Section: model.AdminSectionUsers,
		Action:  model.ActionRead,
	}
	err := repo.Create(ctx, perm)
	assert.NoError(t, err)

	err = repo.Delete(ctx, perm.ID)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, perm.ID)
	assert.Error(t, err)
}

func TestAdminPermissionRepository_FindByID(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	t.Run("find existing permission", func(t *testing.T) {
		perm := &model.AdminPermission{
			RoleID:  role.ID,
			Section: model.AdminSectionUsers,
			Action:  model.ActionRead,
		}
		_ = repo.Create(ctx, perm)

		found, err := repo.FindByID(ctx, perm.ID)
		assert.NoError(t, err)
		assert.Equal(t, perm.ID, found.ID)
	})

	t.Run("find non-existing permission", func(t *testing.T) {
		_, err := repo.FindByID(ctx, 9999)
		assert.Error(t, err)
	})
}

func TestAdminPermissionRepository_FindByRoleID(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()
	role1 := createTestRole(t, db, "role1")
	role2 := createTestRole(t, db, "role2")

	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role1.ID, Section: model.AdminSectionUsers, Action: model.ActionRead})
	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role1.ID, Section: model.AdminSectionRoles, Action: model.ActionWrite})
	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role2.ID, Section: model.AdminSectionUsers, Action: model.ActionRead})

	t.Run("find permissions for role1", func(t *testing.T) {
		perms, err := repo.FindByRoleID(ctx, role1.ID)
		assert.NoError(t, err)
		assert.Len(t, perms, 2)
	})

	t.Run("find permissions for role2", func(t *testing.T) {
		perms, err := repo.FindByRoleID(ctx, role2.ID)
		assert.NoError(t, err)
		assert.Len(t, perms, 1)
	})
}

func TestAdminPermissionRepository_FindByRoleIDs(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()
	role1 := createTestRole(t, db, "role1")
	role2 := createTestRole(t, db, "role2")
	role3 := createTestRole(t, db, "role3")

	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role1.ID, Section: model.AdminSectionUsers, Action: model.ActionRead})
	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role2.ID, Section: model.AdminSectionRoles, Action: model.ActionWrite})
	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role3.ID, Section: model.AdminSectionProjects, Action: model.ActionWrite})

	t.Run("find permissions for multiple roles", func(t *testing.T) {
		perms, err := repo.FindByRoleIDs(ctx, []int64{role1.ID, role2.ID})
		assert.NoError(t, err)
		assert.Len(t, perms, 2)
	})

	t.Run("find permissions with empty role IDs", func(t *testing.T) {
		perms, err := repo.FindByRoleIDs(ctx, []int64{})
		assert.NoError(t, err)
		assert.Len(t, perms, 0)
	})
}

func TestAdminPermissionRepository_FindBySection(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()
	role := createTestRole(t, db, "testrole")

	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role.ID, Section: model.AdminSectionUsers, Action: model.ActionRead})
	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role.ID, Section: model.AdminSectionUsers, Action: model.ActionWrite})
	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role.ID, Section: model.AdminSectionRoles, Action: model.ActionRead})

	t.Run("find by section", func(t *testing.T) {
		perms, err := repo.FindBySection(ctx, model.AdminSectionUsers)
		assert.NoError(t, err)
		assert.Len(t, perms, 2)
	})

	t.Run("find by non-existing section", func(t *testing.T) {
		perms, err := repo.FindBySection(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Len(t, perms, 0)
	})
}

func TestAdminPermissionRepository_DeleteByRoleID(t *testing.T) {
	db := setupPermissionTestDB(t)
	repo := NewAdminPermissionRepository(db)
	ctx := context.Background()
	role1 := createTestRole(t, db, "role1")
	role2 := createTestRole(t, db, "role2")

	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role1.ID, Section: model.AdminSectionUsers, Action: model.ActionRead})
	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role1.ID, Section: model.AdminSectionRoles, Action: model.ActionWrite})
	_ = repo.Create(ctx, &model.AdminPermission{RoleID: role2.ID, Section: model.AdminSectionUsers, Action: model.ActionRead})

	err := repo.DeleteByRoleID(ctx, role1.ID)
	assert.NoError(t, err)

	// role1 permissions should be deleted
	perms, err := repo.FindByRoleID(ctx, role1.ID)
	assert.NoError(t, err)
	assert.Len(t, perms, 0)

	// role2 permissions should remain
	perms, err = repo.FindByRoleID(ctx, role2.ID)
	assert.NoError(t, err)
	assert.Len(t, perms, 1)
}
