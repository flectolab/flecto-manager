package db

import (
	"bytes"
	"errors"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	appContext "github.com/flectolab/flecto-manager/context"
	mockMigratorDB "github.com/flectolab/flecto-manager/mocks/flecto-manager/cli/db"
)

func TestGetMigrateCmd(t *testing.T) {
	ctx := appContext.TestContext(nil)
	cmd := GetMigrateCmd(ctx)

	assert.Equal(t, "migrate", cmd.Use)
	assert.Equal(t, "Database migration commands", cmd.Short)
	assert.Len(t, cmd.Commands(), 3)
}

func TestMigrateApply_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Up().Return(nil)

	ctx := appContext.TestContext(nil)

	// Inject mock
	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateApplyCmd(ctx)
	var out bytes.Buffer
	cmd.SetOut(&out)

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestMigrateApply_NoChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Up().Return(migrate.ErrNoChange)

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateApplyCmd(ctx)

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestMigrateApply_WithSteps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Steps(2).Return(nil)

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateApplyCmd(ctx)
	cmd.SetArgs([]string{"-n", "2"})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestMigrateApply_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Up().Return(errors.New("migration failed"))

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateApplyCmd(ctx)

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to apply migrations")
}

func TestMigrateApply_MigratorCreationError(t *testing.T) {
	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return nil, errors.New("connection failed")
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateApplyCmd(ctx)

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestMigrateStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Version().Return(uint(20260106074436), false, nil)

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateStatusCmd(ctx)

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestMigrateStatus_Dirty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Version().Return(uint(20260106074436), true, nil)

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateStatusCmd(ctx)

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestMigrateStatus_NoMigrations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Version().Return(uint(0), false, migrate.ErrNilVersion)

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateStatusCmd(ctx)

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestMigrateStatus_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Version().Return(uint(0), false, errors.New("db error"))

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateStatusCmd(ctx)

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get version")
}

func TestMigrateStatus_MigratorCreationError(t *testing.T) {
	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return nil, errors.New("connection failed")
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateStatusCmd(ctx)

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestMigrateDown_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Down().Return(nil)

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateDownCmd(ctx)

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestMigrateDown_NoChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Down().Return(migrate.ErrNoChange)

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateDownCmd(ctx)

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestMigrateDown_WithSteps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Steps(-2).Return(nil)

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateDownCmd(ctx)
	cmd.SetArgs([]string{"-n", "2"})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestMigrateDown_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMigrator := mockMigratorDB.NewMockMigrator(ctrl)
	mockMigrator.EXPECT().Down().Return(errors.New("rollback failed"))

	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return mockMigrator, nil
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateDownCmd(ctx)

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to rollback migrations")
}

func TestMigrateDown_MigratorCreationError(t *testing.T) {
	ctx := appContext.TestContext(nil)

	oldNewMigrator := NewMigrator
	NewMigrator = func(ctx *appContext.Context) (Migrator, error) {
		return nil, errors.New("connection failed")
	}
	defer func() { NewMigrator = oldNewMigrator }()

	cmd := getMigrateDownCmd(ctx)

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}