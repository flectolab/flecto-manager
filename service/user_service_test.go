package service

import (
	"context"
	"errors"
	"testing"

	"github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func boolPtr(b bool) *bool {
	return &b
}

func setupUserServiceTest(t *testing.T) (*gomock.Controller, *mockFlectoRepository.MockUserRepository, *mockFlectoRepository.MockRoleRepository, UserService) {
	ctrl := gomock.NewController(t)
	mockUserRepo := mockFlectoRepository.NewMockUserRepository(ctrl)
	mockRoleRepo := mockFlectoRepository.NewMockRoleRepository(ctrl)
	svc := NewUserService(appContext.TestContext(nil), mockUserRepo, mockRoleRepo)
	return ctrl, mockUserRepo, mockRoleRepo, svc
}

func TestNewUserService(t *testing.T) {
	ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockUserRepo)
}

func TestUserService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, mockRoleRepo, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.User{
			Username:  "newuser",
			Firstname: "New",
			Lastname:  "User",
			Active:    boolPtr(true),
		}

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "newuser").
			Return(nil, gorm.ErrRecordNotFound)

		mockUserRepo.EXPECT().
			Create(ctx, input).
			DoAndReturn(func(ctx context.Context, user *model.User) error {
				user.ID = 1
				return nil
			})

		mockRoleRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, role *model.Role) error {
				assert.Equal(t, "newuser", role.Code)
				assert.Equal(t, model.RoleTypeUser, role.Type)
				role.ID = 10
				return nil
			})

		mockRoleRepo.EXPECT().
			AddUserToRole(ctx, int64(1), int64(10)).
			Return(nil)

		result, err := svc.Create(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("user already exists", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.User{
			Username: "existinguser",
			Active:   boolPtr(true),
		}
		existingUser := &model.User{
			ID:       1,
			Username: "existinguser",
		}

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "existinguser").
			Return(existingUser, nil)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrUserAlreadyExists, err)
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.User{
			Username:  "new user",
			Firstname: "John",
			Lastname:  "Doe",
			Active:    boolPtr(true),
		}
		mockUserRepo.EXPECT().
			FindByUsername(ctx, "new user").
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'Username' failed on the 'code' tag")
		assert.Nil(t, result)
	})

	t.Run("repository create error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.User{
			Username:  "newuser",
			Firstname: "John",
			Lastname:  "Doe",
			Active:    boolPtr(true),
		}
		expectedErr := errors.New("database error")

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "newuser").
			Return(nil, gorm.ErrRecordNotFound)

		mockUserRepo.EXPECT().
			Create(ctx, input).
			Return(expectedErr)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("find by username generic error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.User{
			Username: "newuser",
			Active:   boolPtr(true),
		}
		expectedErr := errors.New("database connection error")

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "newuser").
			Return(nil, expectedErr)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("role create error", func(t *testing.T) {
		ctrl, mockUserRepo, mockRoleRepo, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.User{
			Username:  "newuser",
			Firstname: "John",
			Lastname:  "Doe",
			Active:    boolPtr(true),
		}
		expectedErr := errors.New("role create error")

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "newuser").
			Return(nil, gorm.ErrRecordNotFound)

		mockUserRepo.EXPECT().
			Create(ctx, input).
			DoAndReturn(func(ctx context.Context, user *model.User) error {
				user.ID = 1
				return nil
			})

		mockRoleRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(expectedErr)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("add user to role error", func(t *testing.T) {
		ctrl, mockUserRepo, mockRoleRepo, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := &model.User{
			Username:  "newuser",
			Firstname: "John",
			Lastname:  "Doe",
			Active:    boolPtr(true),
		}
		expectedErr := errors.New("add user to role error")

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "newuser").
			Return(nil, gorm.ErrRecordNotFound)

		mockUserRepo.EXPECT().
			Create(ctx, input).
			DoAndReturn(func(ctx context.Context, user *model.User) error {
				user.ID = 1
				return nil
			})

		mockRoleRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, role *model.Role) error {
				role.ID = 10
				return nil
			})

		mockRoleRepo.EXPECT().
			AddUserToRole(ctx, int64(1), int64(10)).
			Return(expectedErr)

		result, err := svc.Create(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestUserService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{
			ID:        1,
			Username:  "testuser",
			Firstname: "Original",
			Lastname:  "Name",
		}
		input := model.User{
			Firstname: "Updated",
			Lastname:  "Person",
		}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockUserRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, user *model.User) error {
				assert.Equal(t, "Updated", user.Firstname)
				assert.Equal(t, "Person", user.Lastname)
				return nil
			})

		result, err := svc.Update(ctx, 1, input)

		assert.NoError(t, err)
		assert.Equal(t, "Updated", result.Firstname)
		assert.Equal(t, "Person", result.Lastname)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := model.User{Firstname: "Updated"}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.Update(ctx, 999, input)

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("invalid data", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{ID: 1, Username: "testuser"}
		input := model.User{Firstname: "Updated"}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		result, err := svc.Update(ctx, 1, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Field validation for 'Lastname' failed on the 'required' tag")
		assert.Nil(t, result)
	})

	t.Run("update error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{ID: 1, Username: "testuser"}
		input := model.User{Firstname: "Updated", Lastname: "Updated"}
		expectedErr := errors.New("update failed")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockUserRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(expectedErr)

		result, err := svc.Update(ctx, 1, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("find by id generic error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		input := model.User{Firstname: "Updated"}
		expectedErr := errors.New("database connection error")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.Update(ctx, 1, input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestUserService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, mockRoleRepo, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{
			ID:       1,
			Username: "testuser",
		}
		existingRole := &model.Role{
			ID:   10,
			Code: "testuser",
			Type: model.RoleTypeUser,
		}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockRoleRepo.EXPECT().
			FindByCodeAndType(ctx, "testuser", model.RoleTypeUser).
			Return(existingRole, nil)

		mockRoleRepo.EXPECT().
			Delete(ctx, int64(10)).
			Return(nil)

		mockUserRepo.EXPECT().
			Delete(ctx, int64(1)).
			Return(nil)

		result, err := svc.Delete(ctx, 1)

		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.Delete(ctx, 1)

		assert.Error(t, err)
		assert.False(t, result)
	})

	t.Run("role not found error", func(t *testing.T) {
		ctrl, mockUserRepo, mockRoleRepo, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{
			ID:       1,
			Username: "testuser",
		}
		expectedErr := errors.New("role not found")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockRoleRepo.EXPECT().
			FindByCodeAndType(ctx, "testuser", model.RoleTypeUser).
			Return(nil, expectedErr)

		result, err := svc.Delete(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})

	t.Run("role delete error", func(t *testing.T) {
		ctrl, mockUserRepo, mockRoleRepo, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{
			ID:       1,
			Username: "testuser",
		}
		existingRole := &model.Role{
			ID:   10,
			Code: "testuser",
			Type: model.RoleTypeUser,
		}
		expectedErr := errors.New("role delete error")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockRoleRepo.EXPECT().
			FindByCodeAndType(ctx, "testuser", model.RoleTypeUser).
			Return(existingRole, nil)

		mockRoleRepo.EXPECT().
			Delete(ctx, int64(10)).
			Return(expectedErr)

		result, err := svc.Delete(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})

	t.Run("user delete error", func(t *testing.T) {
		ctrl, mockUserRepo, mockRoleRepo, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{
			ID:       1,
			Username: "testuser",
		}
		existingRole := &model.Role{
			ID:   10,
			Code: "testuser",
			Type: model.RoleTypeUser,
		}
		expectedErr := errors.New("user delete error")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockRoleRepo.EXPECT().
			FindByCodeAndType(ctx, "testuser", model.RoleTypeUser).
			Return(existingRole, nil)

		mockRoleRepo.EXPECT().
			Delete(ctx, int64(10)).
			Return(nil)

		mockUserRepo.EXPECT().
			Delete(ctx, int64(1)).
			Return(expectedErr)

		result, err := svc.Delete(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.False(t, result)
	})
}

func TestUserService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedUser := &model.User{
			ID:       1,
			Username: "testuser",
		}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(expectedUser, nil)

		result, err := svc.GetByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser, result)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetByID(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("generic error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database connection error")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.GetByID(ctx, 1)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestUserService_GetByUsername(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedUser := &model.User{
			ID:       1,
			Username: "testuser",
		}

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(expectedUser, nil)

		result, err := svc.GetByUsername(ctx, "testuser")

		assert.NoError(t, err)
		assert.Equal(t, expectedUser, result)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "notfound").
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.GetByUsername(ctx, "notfound")

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("generic error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database connection error")

		mockUserRepo.EXPECT().
			FindByUsername(ctx, "testuser").
			Return(nil, expectedErr)

		result, err := svc.GetByUsername(ctx, "testuser")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestUserService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedUsers := []model.User{
			{ID: 1, Username: "user1"},
			{ID: 2, Username: "user2"},
		}

		mockUserRepo.EXPECT().
			FindAll(ctx).
			Return(expectedUsers, nil)

		result, err := svc.GetAll(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expectedUsers, result)
	})

	t.Run("empty result", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockUserRepo.EXPECT().
			FindAll(ctx).
			Return([]model.User{}, nil)

		result, err := svc.GetAll(ctx)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mockUserRepo.EXPECT().
			FindAll(ctx).
			Return(nil, expectedErr)

		result, err := svc.GetAll(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestUserService_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedUsers := []model.User{
			{ID: 1, Username: "user1"},
		}

		mockUserRepo.EXPECT().
			Search(ctx, nil).
			Return(expectedUsers, nil)

		result, err := svc.Search(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, expectedUsers, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("search error")

		mockUserRepo.EXPECT().
			Search(ctx, nil).
			Return(nil, expectedErr)

		result, err := svc.Search(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestUserService_SearchPaginate(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &types.PaginationInput{
			Limit:  &limit,
			Offset: &offset,
		}
		expectedUsers := []model.User{
			{ID: 1, Username: "user1"},
			{ID: 2, Username: "user2"},
		}

		mockUserRepo.EXPECT().
			SearchPaginate(ctx, nil, 10, 5).
			Return(expectedUsers, int64(20), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 20, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 5, result.Offset)
		assert.Len(t, result.Items, 2)
	})

	t.Run("success with default pagination", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedUsers := []model.User{
			{ID: 1, Username: "user1"},
		}

		mockUserRepo.EXPECT().
			SearchPaginate(ctx, nil, types.DefaultLimit, types.DefaultOffset).
			Return(expectedUsers, int64(1), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, types.DefaultLimit, result.Limit)
		assert.Equal(t, types.DefaultOffset, result.Offset)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &types.PaginationInput{}
		expectedErr := errors.New("search error")

		mockUserRepo.EXPECT().
			SearchPaginate(ctx, nil, types.DefaultLimit, types.DefaultOffset).
			Return(nil, int64(0), expectedErr)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestUserService_UpdatePassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockUserRepo.EXPECT().
			UpdatePassword(ctx, int64(1), gomock.Any()).
			Return(nil)

		err := svc.UpdatePassword(ctx, 1, "newpassword")

		assert.NoError(t, err)
	})

	t.Run("repository error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mockUserRepo.EXPECT().
			UpdatePassword(ctx, int64(1), gomock.Any()).
			Return(expectedErr)

		err := svc.UpdatePassword(ctx, 1, "newpassword")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("bcrypt error with too long password", func(t *testing.T) {
		ctrl, _, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Password longer than 72 bytes triggers bcrypt error
		longPassword := string(make([]byte, 73))

		err := svc.UpdatePassword(ctx, 1, longPassword)

		assert.Error(t, err)
	})
}

func TestUserService_UpdateStatus(t *testing.T) {
	t.Run("success deactivate", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{
			ID:       1,
			Username: "testuser",
			Active:   boolPtr(true),
		}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockUserRepo.EXPECT().
			UpdateStatus(ctx, int64(1), false).
			Return(nil)

		result, err := svc.UpdateStatus(ctx, 1, false)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, *result.Active)
	})

	t.Run("success activate", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{
			ID:       1,
			Username: "testuser",
			Active:   boolPtr(false),
		}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockUserRepo.EXPECT().
			UpdateStatus(ctx, int64(1), true).
			Return(nil)

		result, err := svc.UpdateStatus(ctx, 1, true)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, *result.Active)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		result, err := svc.UpdateStatus(ctx, 999, true)

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("update error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{ID: 1, Username: "testuser"}
		expectedErr := errors.New("update failed")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockUserRepo.EXPECT().
			UpdateStatus(ctx, int64(1), true).
			Return(expectedErr)

		result, err := svc.UpdateStatus(ctx, 1, true)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("generic find error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database connection error")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		result, err := svc.UpdateStatus(ctx, 1, true)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestUserService_SetPassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{
			ID:       1,
			Username: "testuser",
		}

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockUserRepo.EXPECT().
			UpdatePassword(ctx, int64(1), gomock.Any()).
			Return(nil)

		err := svc.SetPassword(ctx, 1, "newpassword")

		assert.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(999)).
			Return(nil, gorm.ErrRecordNotFound)

		err := svc.SetPassword(ctx, 999, "newpassword")

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
	})

	t.Run("update error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{ID: 1, Username: "testuser"}
		expectedErr := errors.New("update failed")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		mockUserRepo.EXPECT().
			UpdatePassword(ctx, int64(1), gomock.Any()).
			Return(expectedErr)

		err := svc.SetPassword(ctx, 1, "newpassword")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("generic find error", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database connection error")

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(nil, expectedErr)

		err := svc.SetPassword(ctx, 1, "newpassword")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("bcrypt error with too long password", func(t *testing.T) {
		ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		existingUser := &model.User{
			ID:       1,
			Username: "testuser",
		}

		// Password longer than 72 bytes triggers bcrypt error
		longPassword := string(make([]byte, 73))

		mockUserRepo.EXPECT().
			FindByID(ctx, int64(1)).
			Return(existingUser, nil)

		err := svc.SetPassword(ctx, 1, longPassword)

		assert.Error(t, err)
	})
}

func TestUserService_GetTx(t *testing.T) {
	ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockUserRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestUserService_GetQuery(t *testing.T) {
	ctrl, mockUserRepo, _, svc := setupUserServiceTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockUserRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}
