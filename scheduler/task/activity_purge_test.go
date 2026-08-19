package task

import (
	"context"
	"errors"
	"testing"
	"time"

	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setupActivityPurgeTask(t *testing.T, purgeInterval time.Duration) (*gomock.Controller, *mockFlectoService.MockActivityService, *ActivityPurge, *appContext.Context) {
	ctrl := gomock.NewController(t)
	mockActivity := mockFlectoService.NewMockActivityService(ctrl)

	appCtx := appContext.TestContext(nil)
	appCtx.Config.Activity.PurgeInterval = purgeInterval

	return ctrl, mockActivity, NewActivityPurge(appCtx, mockActivity), appCtx
}

func TestNewActivityPurge(t *testing.T) {
	ctrl, _, task, _ := setupActivityPurgeTask(t, time.Hour)
	defer ctrl.Finish()

	assert.NotNil(t, task)
}

func TestActivityPurge_Name(t *testing.T) {
	ctrl, _, task, _ := setupActivityPurgeTask(t, time.Hour)
	defer ctrl.Finish()

	assert.Equal(t, "activity-purge", task.Name())
}

func TestActivityPurge_Interval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{name: "from config", interval: 30 * time.Minute, want: 30 * time.Minute},
		// A zero interval is how the scheduler is told to skip the task entirely
		{name: "zero disables the task", interval: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, _, task, _ := setupActivityPurgeTask(t, tt.interval)
			defer ctrl.Finish()

			assert.Equal(t, tt.want, task.Interval())
		})
	}
}

func TestActivityPurge_RunOnStart(t *testing.T) {
	ctrl, _, task, _ := setupActivityPurgeTask(t, time.Hour)
	defer ctrl.Finish()

	// An instance that was down while events piled up should not wait a full interval
	assert.True(t, task.RunOnStart())
}

func TestActivityPurge_Timeout(t *testing.T) {
	ctrl, _, task, _ := setupActivityPurgeTask(t, time.Hour)
	defer ctrl.Finish()

	assert.Equal(t, activityPurgeTimeout, task.Timeout())
	assert.Positive(t, task.Timeout())
}

func TestActivityPurge_Run(t *testing.T) {
	t.Run("delegates to the service", func(t *testing.T) {
		ctrl, mockActivity, task, _ := setupActivityPurgeTask(t, time.Hour)
		defer ctrl.Finish()

		ctx := context.Background()
		mockActivity.EXPECT().Purge(ctx).Return(int64(42), nil)

		assert.NoError(t, task.Run(ctx))
	})

	t.Run("nothing purged is not an error", func(t *testing.T) {
		ctrl, mockActivity, task, _ := setupActivityPurgeTask(t, time.Hour)
		defer ctrl.Finish()

		mockActivity.EXPECT().Purge(gomock.Any()).Return(int64(0), nil)

		assert.NoError(t, task.Run(context.Background()))
	})

	t.Run("propagates a failure so the scheduler logs it", func(t *testing.T) {
		ctrl, mockActivity, task, _ := setupActivityPurgeTask(t, time.Hour)
		defer ctrl.Finish()

		errPurge := errors.New("db down")
		mockActivity.EXPECT().Purge(gomock.Any()).Return(int64(0), errPurge)

		assert.ErrorIs(t, task.Run(context.Background()), errPurge)
	})

	t.Run("passes the run context through", func(t *testing.T) {
		ctrl, mockActivity, task, _ := setupActivityPurgeTask(t, time.Hour)
		defer ctrl.Finish()

		// The scheduler cancels this context on shutdown or timeout, so the service
		// must receive it rather than a fresh one
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockActivity.EXPECT().Purge(gomock.Any()).DoAndReturn(func(got context.Context) (int64, error) {
			assert.Equal(t, ctx, got)
			return 0, nil
		})

		assert.NoError(t, task.Run(ctx))
	})
}
