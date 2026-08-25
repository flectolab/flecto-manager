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

func setupCdnPurgeTaskTest(t *testing.T) (*appContext.Context, *mockFlectoService.MockCdnPurgeWorker, *CdnPurge) {
	t.Helper()

	ctrl := gomock.NewController(t)
	ctx := appContext.TestContext(nil)
	worker := mockFlectoService.NewMockCdnPurgeWorker(ctrl)

	return ctx, worker, NewCdnPurge(ctx, worker)
}

func TestCdnPurge_Name(t *testing.T) {
	_, _, task := setupCdnPurgeTaskTest(t)

	assert.Equal(t, "cdn-purge", task.Name())
}

func TestCdnPurge_Interval(t *testing.T) {
	t.Run("successWithTheConfiguredInterval", func(t *testing.T) {
		ctx, _, task := setupCdnPurgeTaskTest(t)
		ctx.Config.Cdn.WorkerInterval = 45 * time.Second

		assert.Equal(t, 45*time.Second, task.Interval())
	})

	t.Run("successIsZeroWhenDisabled", func(t *testing.T) {
		ctx, _, task := setupCdnPurgeTaskTest(t)
		ctx.Config.Cdn.Enabled = false
		ctx.Config.Cdn.WorkerInterval = 45 * time.Second

		// A zero interval is how the scheduler skips the task instead of waking up
		// to do nothing.
		assert.Equal(t, time.Duration(0), task.Interval())
	})
}

func TestCdnPurge_RunOnStart(t *testing.T) {
	_, _, task := setupCdnPurgeTaskTest(t)

	assert.True(t, task.RunOnStart())
}

func TestCdnPurge_Timeout(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{name: "successLeavesAMargin", interval: 30 * time.Second, want: 25 * time.Second},
		{name: "successWithAShortInterval", interval: 3 * time.Second, want: 3 * time.Second},
		{name: "successWhenDisabled", interval: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _, task := setupCdnPurgeTaskTest(t)
			ctx.Config.Cdn.WorkerInterval = tt.interval

			// The timeout has to stay under the interval, otherwise two runs follow
			// each other with no pause.
			assert.Equal(t, tt.want, task.Timeout())
		})
	}
}

func TestCdnPurge_Run(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, worker, task := setupCdnPurgeTaskTest(t)
		ctx := context.Background()

		worker.EXPECT().Run(ctx).Return(2, nil)
		worker.EXPECT().PurgeExpiredJobs(ctx).Return(int64(3), nil)

		assert.NoError(t, task.Run(ctx))
	})

	t.Run("successWithNothingToDo", func(t *testing.T) {
		_, worker, task := setupCdnPurgeTaskTest(t)
		ctx := context.Background()

		worker.EXPECT().Run(ctx).Return(0, nil)
		worker.EXPECT().PurgeExpiredJobs(ctx).Return(int64(0), nil)

		assert.NoError(t, task.Run(ctx))
	})

	t.Run("failWhenSendingFails", func(t *testing.T) {
		_, worker, task := setupCdnPurgeTaskTest(t)
		ctx := context.Background()

		worker.EXPECT().Run(ctx).Return(0, errors.New("boom"))

		// Retention is not even attempted: the run already has a problem to report.
		assert.ErrorContains(t, task.Run(ctx), "boom")
	})

	t.Run("failWhenRetentionFails", func(t *testing.T) {
		_, worker, task := setupCdnPurgeTaskTest(t)
		ctx := context.Background()

		worker.EXPECT().Run(ctx).Return(1, nil)
		worker.EXPECT().PurgeExpiredJobs(ctx).Return(int64(0), errors.New("boom"))

		assert.ErrorContains(t, task.Run(ctx), "boom")
	})
}
