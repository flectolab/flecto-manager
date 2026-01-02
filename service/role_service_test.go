package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type roleServiceMocks struct {
	ctrl     *gomock.Controller
	roleRepo *mockFlectoRepository.MockRoleRepository
	userRepo *mockFlectoRepository.MockUserRepository
}

func setupRoleServiceTest(t *testing.T) (*roleServiceMocks, RoleService) {
	ctrl := gomock.NewController(t)
	mocks := &roleServiceMocks{
		ctrl:     ctrl,
		roleRepo: mockFlectoRepository.NewMockRoleRepository(ctrl),
		userRepo: mockFlectoRepository.NewMockUserRepository(ctrl),
	}
	svc := NewRoleService(appContext.TestContext(nil), mocks.roleRepo, mocks.userRepo)
	return mocks, svc
}

func TestNewRoleService(t *testing.T) {
	mocks, svc := setupRoleServiceTest(t)
	defer mocks.ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mocks.roleRepo)
}

func TestRoleService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		input := &model.Role{
			Code: "newrole",
			Type: model.RoleTypeRole,
		}

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "newrole", model.RoleTypeRole).
			Return(nil, gorm.ErrRecordNotFound)

		mocks.roleRepo.EXPECT().
			Create(ctx, input).
			Return(nil)

		result, err := svc.Create(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("role already exists", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		input := &model.Role{
			Code: "existingrole",
			Type: model.RoleTypeRole,
		}
		existingRole := &model.Role{
			ID:   1,
			Code: "existingrole",
		}

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "existingrole", model.RoleTypeRole).
			Return(existingRole, nil)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleAlreadyExists, err)
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		input := &model.Role{Code: "new role", Type: model.RoleTypeRole}

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "new role", model.RoleTypeRole).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'Code' failed on the 'code' tag")
		assert.Nil(t, result)
	})

	t.Run("repository create error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		input := &model.Role{Code: "newrole", Type: model.RoleTypeRole}
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "newrole", model.RoleTypeRole).
			Return(nil, gorm.ErrRecordNotFound)

		mocks.roleRepo.EXPECT().
			Create(ctx, input).
			Return(expectedErr)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("find by code generic error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		input := &model.Role{Code: "newrole", Type: model.RoleTypeRole}
		expectedErr := errors.New("database connection error")

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "newrole", model.RoleTypeRole).
			Return(nil, expectedErr)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestRoleService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		existingRole := &model.Role{
			ID:   1,
			Code: "oldrole",
			Type: model.RoleTypeRole,
		}
		input := model.Role{
			Code: "updatedrole",
			Type: model.RoleTypeRole,
		}

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingRole, nil)

		mocks.roleRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, role *model.Role) error {
				assert.Equal(t, "updatedrole", role.Code)
				return nil
			})

		result, err := svc.Update(ctx, 1, input)

		assert.NoError(t, err)
		assert.Equal(t, "updatedrole", result.Code)
	})

	t.Run("role not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		input := model.Role{Code: "updated"}

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.Update(ctx, 999, input)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		existingRole := &model.Role{ID: 1, Code: "testrole"}
		input := model.Role{Code: "updated wrong"}

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingRole, nil)

		result, err := svc.Update(ctx, 1, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'Code' failed on the 'code' tag")
		assert.Nil(t, result)
	})

	t.Run("update error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		existingRole := &model.Role{ID: 1, Code: "testrole"}
		input := model.Role{Code: "updated"}
		expectedErr := errors.New("update failed")

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingRole, nil)

		mocks.roleRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(expectedErr)

		result, err := svc.Update(ctx, 1, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("find by id generic error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		input := model.Role{Code: "updated"}
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.Update(ctx, 1, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestRoleService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		existingRole := &model.Role{ID: 1, Code: "todelete"}

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingRole, nil)

		mocks.roleRepo.EXPECT().
			Delete(ctx, int64(1)).
			Return(nil)

		result, err := svc.Delete(ctx, 1)

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("role not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.Delete(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
		assert.False(t, result)
	})

	t.Run("delete error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		existingRole := &model.Role{ID: 1, Code: "todelete"}
		expectedErr := errors.New("delete failed")

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingRole, nil)

		mocks.roleRepo.EXPECT().
			Delete(ctx, int64(1)).
			Return(expectedErr)

		result, err := svc.Delete(ctx, 1)

		assert.Error(t, err)
		assert.False(t, result)
	})

	t.Run("find by id generic error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.Delete(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})
}

func TestRoleService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedRole := &model.Role{
			ID:   1,
			Code: "testrole",
		}

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(expectedRole, nil)

		result, err := svc.GetByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedRole, result)
	})

	t.Run("not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetByID(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("generic error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.GetByID(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestRoleService_GetByCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedRole := &model.Role{
			ID:   1,
			Code: "testrole",
			Type: model.RoleTypeRole,
		}

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "testrole", model.RoleTypeRole).
			Return(expectedRole, nil)

		result, err := svc.GetByCode(ctx, "testrole", model.RoleTypeRole)

		assert.NoError(t, err)
		assert.Equal(t, expectedRole, result)
	})

	t.Run("success with user type", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedRole := &model.Role{
			ID:   1,
			Code: "john.doe",
			Type: model.RoleTypeUser,
		}

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "john.doe", model.RoleTypeUser).
			Return(expectedRole, nil)

		result, err := svc.GetByCode(ctx, "john.doe", model.RoleTypeUser)

		assert.NoError(t, err)
		assert.Equal(t, expectedRole, result)
	})

	t.Run("not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "notfound", model.RoleTypeRole).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetByCode(ctx, "notfound", model.RoleTypeRole)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("generic error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "testrole", model.RoleTypeRole).
			Return(nil, expectedErr)

		result, err := svc.GetByCode(ctx, "testrole", model.RoleTypeRole)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestRoleService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedRoles := []model.Role{
			{ID: 1, Code: "role1"},
			{ID: 2, Code: "role2"},
		}

		mocks.roleRepo.EXPECT().
			FindAll(ctx).
			Return(expectedRoles, nil)

		result, err := svc.GetAll(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expectedRoles, result)
	})

	t.Run("empty result", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			FindAll(ctx).
			Return([]model.Role{}, nil)

		result, err := svc.GetAll(ctx)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestRoleService_GetAllByType(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedRoles := []model.Role{
			{ID: 1, Code: "role1", Type: model.RoleTypeRole},
			{ID: 2, Code: "role2", Type: model.RoleTypeRole},
		}

		mocks.roleRepo.EXPECT().
			FindAllByType(ctx, model.RoleTypeRole).
			Return(expectedRoles, nil)

		result, err := svc.GetAllByType(ctx, model.RoleTypeRole)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestRoleService_SearchPaginate(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &types.PaginationInput{
			Limit:  &limit,
			Offset: &offset,
		}
		expectedRoles := []model.Role{
			{ID: 1, Code: "role1"},
			{ID: 2, Code: "role2"},
		}

		mocks.roleRepo.EXPECT().
			SearchPaginate(ctx, nil, 10, 5).
			Return(expectedRoles, int64(20), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 20, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 5, result.Offset)
		assert.Len(t, result.Items, 2)
	})

	t.Run("error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedErr := errors.New("search error")

		mocks.roleRepo.EXPECT().
			SearchPaginate(ctx, nil, types.DefaultLimit, types.DefaultOffset).
			Return(nil, int64(0), expectedErr)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestRoleService_AddUserToRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		role := &model.Role{ID: 1, Code: "testrole"}

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(role, nil)

		mocks.roleRepo.EXPECT().
			HasUserRole(ctx, int64(10), int64(1)).
			Return(false, nil)

		mocks.roleRepo.EXPECT().
			AddUserToRole(ctx, int64(10), int64(1)).
			Return(nil)

		err := svc.AddUserToRole(ctx, 10, 1)

		assert.NoError(t, err)
	})

	t.Run("role not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		err := svc.AddUserToRole(ctx, 10, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})

	t.Run("user already in role", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		role := &model.Role{ID: 1, Code: "testrole"}

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(role, nil)

		mocks.roleRepo.EXPECT().
			HasUserRole(ctx, int64(10), int64(1)).
			Return(true, nil)

		err := svc.AddUserToRole(ctx, 10, 1)

		assert.Error(t, err)
		assert.Equal(t, ErrUserAlreadyInRole, err)
	})

	t.Run("find by id generic error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		err := svc.AddUserToRole(ctx, 10, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("has user role error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		role := &model.Role{ID: 1, Code: "testrole"}
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(role, nil)

		mocks.roleRepo.EXPECT().
			HasUserRole(ctx, int64(10), int64(1)).
			Return(false, expectedErr)

		err := svc.AddUserToRole(ctx, 10, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestRoleService_RemoveUserFromRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			HasUserRole(ctx, int64(10), int64(1)).
			Return(true, nil)

		mocks.roleRepo.EXPECT().
			RemoveUserFromRole(ctx, int64(10), int64(1)).
			Return(nil)

		err := svc.RemoveUserFromRole(ctx, 10, 1)

		assert.NoError(t, err)
	})

	t.Run("user not in role", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			HasUserRole(ctx, int64(10), int64(1)).
			Return(false, nil)

		err := svc.RemoveUserFromRole(ctx, 10, 1)

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotInRole, err)
	})

	t.Run("has user role error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			HasUserRole(ctx, int64(10), int64(1)).
			Return(false, expectedErr)

		err := svc.RemoveUserFromRole(ctx, 10, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestRoleService_GetUserRoles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedRoles := []model.Role{
			{ID: 1, Code: "role1"},
			{ID: 2, Code: "role2"},
		}

		mocks.roleRepo.EXPECT().
			GetUserRoles(ctx, int64(10)).
			Return(expectedRoles, nil)

		result, err := svc.GetUserRoles(ctx, 10)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestRoleService_GetUserRolesByType(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedRoles := []model.Role{
			{ID: 1, Code: "role1", Type: model.RoleTypeRole},
			{ID: 2, Code: "role2", Type: model.RoleTypeRole},
		}

		mocks.roleRepo.EXPECT().
			GetUserRolesByType(ctx, int64(10), model.RoleTypeRole).
			Return(expectedRoles, nil)

		result, err := svc.GetUserRolesByType(ctx, 10, model.RoleTypeRole)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestRoleService_GetRoleUsers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		role := &model.Role{ID: 1, Code: "testrole"}
		expectedUsers := []model.User{
			{ID: 1, Username: "user1"},
			{ID: 2, Username: "user2"},
		}

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(role, nil)

		mocks.roleRepo.EXPECT().
			GetRoleUsers(ctx, int64(1)).
			Return(expectedUsers, nil)

		result, err := svc.GetRoleUsers(ctx, 1)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("role not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetRoleUsers(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("find by id generic error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.GetRoleUsers(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestRoleService_GetPermissionsByRoleCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		// Permissions are now preloaded in the role
		role := &model.Role{
			ID:   1,
			Code: "admin",
			Resources: []model.ResourcePermission{
				{ID: 1, Namespace: "ns1", Project: "proj1", Action: model.ActionRead, RoleID: 1},
			},
			Admin: []model.AdminPermission{
				{ID: 1, Section: model.AdminSectionUsers, Action: model.ActionRead, RoleID: 1},
			},
		}

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "admin", model.RoleTypeRole).
			Return(role, nil)

		result, err := svc.GetPermissionsByRoleCode(ctx, "admin")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Resources, 1)
		assert.Len(t, result.Admin, 1)
	})

	t.Run("role not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "notfound", model.RoleTypeRole).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetPermissionsByRoleCode(ctx, "notfound")

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("generic error from FindByCodeAndType", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "admin", model.RoleTypeRole).
			Return(nil, expectedErr)

		result, err := svc.GetPermissionsByRoleCode(ctx, "admin")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestRoleService_GetPermissionsByUsername(t *testing.T) {
	t.Run("success with multiple roles and deduplication", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		user := &model.User{ID: 1, Username: "testuser"}
		// Permissions are now preloaded in the roles
		roles := []model.Role{
			{
				ID:   1,
				Code: "role1",
				Resources: []model.ResourcePermission{
					{ID: 1, Namespace: "ns1", Project: "proj1", Action: model.ActionRead, RoleID: 1},
				},
				Admin: []model.AdminPermission{
					{ID: 1, Section: model.AdminSectionUsers, Action: model.ActionRead, RoleID: 1},
				},
			},
			{
				ID:   2,
				Code: "role2",
				Resources: []model.ResourcePermission{
					{ID: 2, Namespace: "ns1", Project: "proj1", Action: model.ActionRead, RoleID: 2}, // Duplicate
					{ID: 3, Namespace: "ns2", Project: "proj2", Action: model.ActionWrite, RoleID: 2},
				},
				Admin: []model.AdminPermission{
					{ID: 2, Section: model.AdminSectionUsers, Action: model.ActionRead, RoleID: 2}, // Duplicate
					{ID: 3, Section: model.AdminSectionRoles, Action: model.ActionWrite, RoleID: 2},
				},
			},
		}

		mocks.userRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(user, nil)

		mocks.roleRepo.EXPECT().
			GetUserRoles(ctx, int64(1)).
			Return(roles, nil)

		result, err := svc.GetPermissionsByUsername(ctx, "testuser")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Resources, 2) // Deduplicated
		assert.Len(t, result.Admin, 2)     // Deduplicated
	})

	t.Run("user not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.userRepo.EXPECT().
			FindByUsername(ctx, "notfound").
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetPermissionsByUsername(ctx, "notfound")

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("generic error from FindByUsername", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.userRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(nil, expectedErr)

		result, err := svc.GetPermissionsByUsername(ctx, "testuser")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("user with no roles", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		user := &model.User{ID: 1, Username: "noroles"}

		mocks.userRepo.EXPECT().
			FindByUsername(ctx, "noroles").
			Return(user, nil)

		mocks.roleRepo.EXPECT().
			GetUserRoles(ctx, int64(1)).
			Return([]model.Role{}, nil)

		result, err := svc.GetPermissionsByUsername(ctx, "noroles")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Resources)
		assert.Empty(t, result.Admin)
	})

	t.Run("error from GetUserRoles", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		user := &model.User{ID: 1, Username: "testuser"}
		expectedErr := errors.New("roles fetch error")

		mocks.userRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(user, nil)

		mocks.roleRepo.EXPECT().
			GetUserRoles(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.GetPermissionsByUsername(ctx, "testuser")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestRoleService_GetPermissionsByTokenName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		role := &model.Role{
			ID:   1,
			Code: "token_mytoken",
			Type: model.RoleTypeToken,
			Resources: []model.ResourcePermission{
				{ID: 1, Namespace: "ns1", Project: "proj1", Action: model.ActionRead, RoleID: 1},
			},
			Admin: []model.AdminPermission{
				{ID: 1, Section: model.AdminSectionUsers, Action: model.ActionRead, RoleID: 1},
			},
		}

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "token_mytoken", model.RoleTypeToken).
			Return(role, nil)

		result, err := svc.GetPermissionsByTokenName(ctx, "mytoken")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Resources, 1)
		assert.Len(t, result.Admin, 1)
	})

	t.Run("role not found returns empty permissions", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "token_unknown", model.RoleTypeToken).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetPermissionsByTokenName(ctx, "unknown")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Resources)
		assert.Empty(t, result.Admin)
	})

	t.Run("generic error", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "token_mytoken", model.RoleTypeToken).
			Return(nil, expectedErr)

		result, err := svc.GetPermissionsByTokenName(ctx, "mytoken")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestDeduplicateResourcePermissions(t *testing.T) {
	tests := []struct {
		name     string
		input    []model.ResourcePermission
		expected int
	}{
		{
			name:     "empty input",
			input:    []model.ResourcePermission{},
			expected: 0,
		},
		{
			name: "no duplicates",
			input: []model.ResourcePermission{
				{Namespace: "ns1", Project: "p1", Resource: model.ResourceTypeRedirect, Action: model.ActionRead},
				{Namespace: "ns2", Project: "p2", Resource: model.ResourceTypePage, Action: model.ActionWrite},
			},
			expected: 2,
		},
		{
			name: "with duplicates",
			input: []model.ResourcePermission{
				{Namespace: "ns1", Project: "p1", Resource: model.ResourceTypeRedirect, Action: model.ActionRead},
				{Namespace: "ns1", Project: "p1", Resource: model.ResourceTypeRedirect, Action: model.ActionRead},
				{Namespace: "ns2", Project: "p2", Resource: model.ResourceTypePage, Action: model.ActionWrite},
			},
			expected: 2,
		},
		{
			name: "same namespace/project/action but different resource",
			input: []model.ResourcePermission{
				{Namespace: "ns1", Project: "p1", Resource: model.ResourceTypeRedirect, Action: model.ActionRead},
				{Namespace: "ns1", Project: "p1", Resource: model.ResourceTypePage, Action: model.ActionRead},
				{Namespace: "ns1", Project: "p1", Resource: model.ResourceTypeAgent, Action: model.ActionRead},
			},
			expected: 3,
		},
		{
			name: "wildcard resource with duplicates",
			input: []model.ResourcePermission{
				{Namespace: "ns1", Project: "p1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
				{Namespace: "ns1", Project: "p1", Resource: model.ResourceTypeAll, Action: model.ActionRead},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateResourcePermissions(tt.input)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestDeduplicateAdminPermissions(t *testing.T) {
	tests := []struct {
		name     string
		input    []model.AdminPermission
		expected int
	}{
		{
			name:     "empty input",
			input:    []model.AdminPermission{},
			expected: 0,
		},
		{
			name: "no duplicates",
			input: []model.AdminPermission{
				{Section: model.AdminSectionUsers, Action: model.ActionRead},
				{Section: model.AdminSectionRoles, Action: model.ActionWrite},
			},
			expected: 2,
		},
		{
			name: "with duplicates",
			input: []model.AdminPermission{
				{Section: model.AdminSectionUsers, Action: model.ActionRead},
				{Section: model.AdminSectionUsers, Action: model.ActionRead},
				{Section: model.AdminSectionRoles, Action: model.ActionWrite},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateAdminPermissions(tt.input)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestRoleService_UpdateRolePermissions(t *testing.T) {
	t.Run("role not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{},
			Admin:     []model.AdminPermission{},
		}

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		err := svc.UpdateRolePermissions(ctx, 999, permissions)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})

	t.Run("generic error from FindByID", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		permissions := &model.SubjectPermissions{}
		expectedErr := errors.New("database error")

		mocks.roleRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		err := svc.UpdateRolePermissions(ctx, 1, permissions)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

// Integration tests for UpdateRolePermissions using SQLite in-memory

func setupRoleServiceIntegrationTest(t *testing.T) (*gorm.DB, RoleService) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Role{}, &model.User{}, &model.ResourcePermission{}, &model.AdminPermission{})
	assert.NoError(t, err)

	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	svc := NewRoleService(appContext.TestContext(nil), roleRepo, userRepo)
	return db, svc
}

func TestRoleService_UpdateRolePermissions_Integration(t *testing.T) {
	t.Run("success - replace all permissions", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTest(t)
		ctx := context.Background()

		// Create a role with a specific updatedAt time in the past
		initialTime := time.Now().Add(-1 * time.Hour)
		role := &model.Role{Code: "testrole", Type: model.RoleTypeRole, UpdatedAt: initialTime}
		err := db.Create(role).Error
		assert.NoError(t, err)

		// Add initial permissions
		err = db.Create(&model.ResourcePermission{RoleID: role.ID, Namespace: "old-ns", Project: "old-proj", Action: model.ActionRead}).Error
		assert.NoError(t, err)
		err = db.Create(&model.AdminPermission{RoleID: role.ID, Section: model.AdminSectionUsers, Action: model.ActionRead}).Error
		assert.NoError(t, err)

		// Update with new permissions
		newPermissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "new-ns1", Project: "new-proj1", Action: model.ActionWrite},
				{Namespace: "new-ns2", Project: "new-proj2", Action: model.ActionWrite},
			},
			Admin: []model.AdminPermission{
				{Section: model.AdminSectionRoles, Action: model.ActionWrite},
			},
		}

		err = svc.UpdateRolePermissions(ctx, role.ID, newPermissions)
		assert.NoError(t, err)

		// Verify old permissions are deleted and new ones are created
		var resources []model.ResourcePermission
		err = db.Where("role_id = ?", role.ID).Find(&resources).Error
		assert.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "new-ns1", resources[0].Namespace)
		assert.Equal(t, "new-ns2", resources[1].Namespace)

		var admin []model.AdminPermission
		err = db.Where("role_id = ?", role.ID).Find(&admin).Error
		assert.NoError(t, err)
		assert.Len(t, admin, 1)
		assert.Equal(t, model.AdminSectionRoles, admin[0].Section)

		// Verify role's updatedAt was updated
		var updatedRole model.Role
		err = db.First(&updatedRole, role.ID).Error
		assert.NoError(t, err)
		assert.True(t, updatedRole.UpdatedAt.After(initialTime), "role updatedAt should be updated after permission change")
	})

	t.Run("success - clear all permissions with empty input", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTest(t)
		ctx := context.Background()

		// Create a role
		role := &model.Role{Code: "testrole", Type: model.RoleTypeRole}
		err := db.Create(role).Error
		assert.NoError(t, err)

		// Add initial permissions
		err = db.Create(&model.ResourcePermission{RoleID: role.ID, Namespace: "ns", Project: "proj", Action: model.ActionRead}).Error
		assert.NoError(t, err)
		err = db.Create(&model.AdminPermission{RoleID: role.ID, Section: model.AdminSectionUsers, Action: model.ActionRead}).Error
		assert.NoError(t, err)

		// Clear all permissions
		emptyPermissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{},
			Admin:     []model.AdminPermission{},
		}

		err = svc.UpdateRolePermissions(ctx, role.ID, emptyPermissions)
		assert.NoError(t, err)

		// Verify all permissions are deleted
		var resourceCount int64
		err = db.Model(&model.ResourcePermission{}).Where("role_id = ?", role.ID).Count(&resourceCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), resourceCount)

		var adminCount int64
		err = db.Model(&model.AdminPermission{}).Where("role_id = ?", role.ID).Count(&adminCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), adminCount)
	})

	t.Run("success - add permissions to role without existing permissions", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTest(t)
		ctx := context.Background()

		// Create a role without permissions
		role := &model.Role{Code: "newrole", Type: model.RoleTypeRole}
		err := db.Create(role).Error
		assert.NoError(t, err)

		// Add permissions
		newPermissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
			},
			Admin: []model.AdminPermission{
				{Section: model.AdminSectionUsers, Action: model.ActionWrite},
			},
		}

		err = svc.UpdateRolePermissions(ctx, role.ID, newPermissions)
		assert.NoError(t, err)

		// Verify permissions are created
		var resources []model.ResourcePermission
		err = db.Where("role_id = ?", role.ID).Find(&resources).Error
		assert.NoError(t, err)
		assert.Len(t, resources, 1)

		var admin []model.AdminPermission
		err = db.Where("role_id = ?", role.ID).Find(&admin).Error
		assert.NoError(t, err)
		assert.Len(t, admin, 1)
	})

	t.Run("does not affect other roles permissions", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTest(t)
		ctx := context.Background()

		// Create two roles
		role1 := &model.Role{Code: "role1", Type: model.RoleTypeRole}
		err := db.Create(role1).Error
		assert.NoError(t, err)

		role2 := &model.Role{Code: "role2", Type: model.RoleTypeRole}
		err = db.Create(role2).Error
		assert.NoError(t, err)

		// Add permissions to both roles
		err = db.Create(&model.ResourcePermission{RoleID: role1.ID, Namespace: "ns1", Project: "proj1", Action: model.ActionRead}).Error
		assert.NoError(t, err)
		err = db.Create(&model.ResourcePermission{RoleID: role2.ID, Namespace: "ns2", Project: "proj2", Action: model.ActionWrite}).Error
		assert.NoError(t, err)

		// Update role1 permissions
		newPermissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "updated-ns", Project: "updated-proj", Action: model.ActionWrite},
			},
			Admin: []model.AdminPermission{},
		}

		err = svc.UpdateRolePermissions(ctx, role1.ID, newPermissions)
		assert.NoError(t, err)

		// Verify role2 permissions are unchanged
		var role2Resources []model.ResourcePermission
		err = db.Where("role_id = ?", role2.ID).Find(&role2Resources).Error
		assert.NoError(t, err)
		assert.Len(t, role2Resources, 1)
		assert.Equal(t, "ns2", role2Resources[0].Namespace)
	})

	t.Run("error - delete resource permissions fails", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTest(t)
		ctx := context.Background()

		// Create a role
		role := &model.Role{Code: "testrole", Type: model.RoleTypeRole}
		err := db.Create(role).Error
		assert.NoError(t, err)

		// Drop the resource_permissions table to cause an error
		err = db.Exec("DROP TABLE resource_permissions").Error
		assert.NoError(t, err)

		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{},
			Admin:     []model.AdminPermission{},
		}

		err = svc.UpdateRolePermissions(ctx, role.ID, permissions)
		assert.Error(t, err)
	})

	t.Run("error - delete admin permissions fails", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTest(t)
		ctx := context.Background()

		// Create a role
		role := &model.Role{Code: "testrole", Type: model.RoleTypeRole}
		err := db.Create(role).Error
		assert.NoError(t, err)

		// Drop the admin_permissions table to cause an error
		err = db.Exec("DROP TABLE admin_permissions").Error
		assert.NoError(t, err)

		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{},
			Admin:     []model.AdminPermission{},
		}

		err = svc.UpdateRolePermissions(ctx, role.ID, permissions)
		assert.Error(t, err)
	})

	t.Run("error - create resource permissions fails", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTest(t)
		ctx := context.Background()

		// Create a role
		role := &model.Role{Code: "testrole", Type: model.RoleTypeRole}
		err := db.Create(role).Error
		assert.NoError(t, err)

		// Add a callback to drop the table after delete but before create
		db.Callback().Create().Before("gorm:create").Register("drop_resource_table", func(tx *gorm.DB) {
			if tx.Statement.Table == "resource_permissions" {
				tx.Exec("DROP TABLE resource_permissions")
			}
		})

		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{
				{Namespace: "ns1", Project: "proj1", Action: model.ActionRead},
			},
			Admin: []model.AdminPermission{},
		}

		err = svc.UpdateRolePermissions(ctx, role.ID, permissions)
		assert.Error(t, err)
	})

	t.Run("error - create admin permissions fails", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTest(t)
		ctx := context.Background()

		// Create a role
		role := &model.Role{Code: "testrole", Type: model.RoleTypeRole}
		err := db.Create(role).Error
		assert.NoError(t, err)

		// Add a callback to drop the table after delete but before create
		db.Callback().Create().Before("gorm:create").Register("drop_admin_table", func(tx *gorm.DB) {
			if tx.Statement.Table == "admin_permissions" {
				tx.Exec("DROP TABLE admin_permissions")
			}
		})

		permissions := &model.SubjectPermissions{
			Resources: []model.ResourcePermission{},
			Admin: []model.AdminPermission{
				{Section: model.AdminSectionUsers, Action: model.ActionRead},
			},
		}

		err = svc.UpdateRolePermissions(ctx, role.ID, permissions)
		assert.Error(t, err)
	})
}

func TestRoleService_UpdateUserRoles(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()

		mocks.userRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		err := svc.UpdateUserRoles(ctx, 999, []string{"role1"})

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
	})

	t.Run("generic error from FindByID", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mocks.userRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		err := svc.UpdateUserRoles(ctx, 1, []string{"role1"})

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("role not found during code resolution", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		user := &model.User{ID: 1, Username: "testuser"}

		mocks.userRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(user, nil)

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "unknownrole", model.RoleTypeRole).
			Return(nil, gorm.ErrRecordNotFound)

		err := svc.UpdateUserRoles(ctx, 1, []string{"unknownrole"})

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})

	t.Run("generic error from FindByCodeAndType", func(t *testing.T) {
		mocks, svc := setupRoleServiceTest(t)
		defer mocks.ctrl.Finish()

		ctx := context.Background()
		user := &model.User{ID: 1, Username: "testuser"}
		expectedErr := errors.New("database error")

		mocks.userRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(user, nil)

		mocks.roleRepo.EXPECT().
			FindByCodeAndType(ctx, "role1", model.RoleTypeRole).
			Return(nil, expectedErr)

		err := svc.UpdateUserRoles(ctx, 1, []string{"role1"})

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func setupRoleServiceIntegrationTestWithUserRoles(t *testing.T) (*gorm.DB, RoleService) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Role{}, &model.User{}, &model.UserRole{}, &model.ResourcePermission{}, &model.AdminPermission{})
	assert.NoError(t, err)

	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	svc := NewRoleService(appContext.TestContext(nil), roleRepo, userRepo)
	return db, svc
}

func TestRoleService_UpdateUserRoles_Integration(t *testing.T) {
	t.Run("success - replace existing roles with new roles", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTestWithUserRoles(t)
		ctx := context.Background()

		// Create a user
		user := &model.User{Username: "testuser", Password: "test"}
		err := db.Create(user).Error
		assert.NoError(t, err)

		// Create roles
		role1 := &model.Role{Code: "role1", Type: model.RoleTypeRole}
		role2 := &model.Role{Code: "role2", Type: model.RoleTypeRole}
		role3 := &model.Role{Code: "role3", Type: model.RoleTypeRole}
		err = db.Create(role1).Error
		assert.NoError(t, err)
		err = db.Create(role2).Error
		assert.NoError(t, err)
		err = db.Create(role3).Error
		assert.NoError(t, err)

		// Add user to role1 initially
		err = db.Create(&model.UserRole{UserID: user.ID, RoleID: role1.ID}).Error
		assert.NoError(t, err)

		// Update user roles to role2 and role3
		err = svc.UpdateUserRoles(ctx, user.ID, []string{"role2", "role3"})
		assert.NoError(t, err)

		// Verify user now has role2 and role3, not role1
		var userRoles []model.UserRole
		err = db.Where("user_id = ?", user.ID).Find(&userRoles).Error
		assert.NoError(t, err)
		assert.Len(t, userRoles, 2)

		roleIDs := []int64{userRoles[0].RoleID, userRoles[1].RoleID}
		assert.Contains(t, roleIDs, role2.ID)
		assert.Contains(t, roleIDs, role3.ID)
		assert.NotContains(t, roleIDs, role1.ID)
	})

	t.Run("success - remove all roles with empty list", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTestWithUserRoles(t)
		ctx := context.Background()

		// Create a user
		user := &model.User{Username: "testuser", Password: "test"}
		err := db.Create(user).Error
		assert.NoError(t, err)

		// Create and assign a role
		role := &model.Role{Code: "role1", Type: model.RoleTypeRole}
		err = db.Create(role).Error
		assert.NoError(t, err)
		err = db.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error
		assert.NoError(t, err)

		// Remove all roles
		err = svc.UpdateUserRoles(ctx, user.ID, []string{})
		assert.NoError(t, err)

		// Verify user has no roles
		var count int64
		err = db.Model(&model.UserRole{}).Where("user_id = ?", user.ID).Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("success - add roles to user without existing roles", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTestWithUserRoles(t)
		ctx := context.Background()

		// Create a user without roles
		user := &model.User{Username: "newuser", Password: "test"}
		err := db.Create(user).Error
		assert.NoError(t, err)

		// Create roles
		role1 := &model.Role{Code: "role1", Type: model.RoleTypeRole}
		role2 := &model.Role{Code: "role2", Type: model.RoleTypeRole}
		err = db.Create(role1).Error
		assert.NoError(t, err)
		err = db.Create(role2).Error
		assert.NoError(t, err)

		// Add roles
		err = svc.UpdateUserRoles(ctx, user.ID, []string{"role1", "role2"})
		assert.NoError(t, err)

		// Verify user has the roles
		var userRoles []model.UserRole
		err = db.Where("user_id = ?", user.ID).Find(&userRoles).Error
		assert.NoError(t, err)
		assert.Len(t, userRoles, 2)
	})

	t.Run("does not affect other users roles", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTestWithUserRoles(t)
		ctx := context.Background()

		// Create two users
		user1 := &model.User{Username: "user1", Password: "test"}
		user2 := &model.User{Username: "user2", Password: "test"}
		err := db.Create(user1).Error
		assert.NoError(t, err)
		err = db.Create(user2).Error
		assert.NoError(t, err)

		// Create roles
		role1 := &model.Role{Code: "role1", Type: model.RoleTypeRole}
		role2 := &model.Role{Code: "role2", Type: model.RoleTypeRole}
		err = db.Create(role1).Error
		assert.NoError(t, err)
		err = db.Create(role2).Error
		assert.NoError(t, err)

		// Assign role1 to both users
		err = db.Create(&model.UserRole{UserID: user1.ID, RoleID: role1.ID}).Error
		assert.NoError(t, err)
		err = db.Create(&model.UserRole{UserID: user2.ID, RoleID: role1.ID}).Error
		assert.NoError(t, err)

		// Update user1 roles to role2 only
		err = svc.UpdateUserRoles(ctx, user1.ID, []string{"role2"})
		assert.NoError(t, err)

		// Verify user2 still has role1
		var user2Roles []model.UserRole
		err = db.Where("user_id = ?", user2.ID).Find(&user2Roles).Error
		assert.NoError(t, err)
		assert.Len(t, user2Roles, 1)
		assert.Equal(t, role1.ID, user2Roles[0].RoleID)
	})

	t.Run("error - delete user_roles fails", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTestWithUserRoles(t)
		ctx := context.Background()

		// Create a user
		user := &model.User{Username: "testuser", Password: "test"}
		err := db.Create(user).Error
		assert.NoError(t, err)

		// Create a role
		role := &model.Role{Code: "role1", Type: model.RoleTypeRole}
		err = db.Create(role).Error
		assert.NoError(t, err)

		// Drop the user_roles table to cause an error
		err = db.Exec("DROP TABLE user_roles").Error
		assert.NoError(t, err)

		err = svc.UpdateUserRoles(ctx, user.ID, []string{"role1"})
		assert.Error(t, err)
	})

	t.Run("error - create user_roles fails", func(t *testing.T) {
		db, svc := setupRoleServiceIntegrationTestWithUserRoles(t)
		ctx := context.Background()

		// Create a user
		user := &model.User{Username: "testuser", Password: "test"}
		err := db.Create(user).Error
		assert.NoError(t, err)

		// Create a role
		role := &model.Role{Code: "role1", Type: model.RoleTypeRole}
		err = db.Create(role).Error
		assert.NoError(t, err)

		// Add a callback to drop the table after delete but before create
		db.Callback().Create().Before("gorm:create").Register("drop_user_roles_table", func(tx *gorm.DB) {
			if tx.Statement.Table == "user_roles" {
				tx.Exec("DROP TABLE user_roles")
			}
		})

		err = svc.UpdateUserRoles(ctx, user.ID, []string{"role1"})
		assert.Error(t, err)
	})
}

func TestRoleService_GetTx(t *testing.T) {
	mocks, svc := setupRoleServiceTest(t)
	defer mocks.ctrl.Finish()

	ctx := context.Background()
	mocks.roleRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestRoleService_GetQuery(t *testing.T) {
	mocks, svc := setupRoleServiceTest(t)
	defer mocks.ctrl.Finish()

	ctx := context.Background()
	mocks.roleRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}
