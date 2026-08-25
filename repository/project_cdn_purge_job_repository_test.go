package repository

import (
	"context"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/cdn"
	"github.com/flectolab/flecto-manager/crypt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProjectCdnPurgeJobTestDB(t *testing.T) (*gorm.DB, int64) {
	t.Helper()

	key, err := crypt.GenerateKey()
	assert.NoError(t, err)
	cipher, err := crypt.NewAESGCM(key)
	assert.NoError(t, err)
	crypt.RegisterSerializer(cipher)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(
		&model.Namespace{}, &model.Project{}, &model.ProjectCdnConfig{}, &model.ProjectCdnPurgeJob{},
	))

	assert.NoError(t, db.Create(&model.Namespace{NamespaceCode: "ns1", Name: "ns1"}).Error)
	assert.NoError(t, db.Create(&model.Project{NamespaceCode: "ns1", ProjectCode: "proj1", Name: "proj1"}).Error)

	config := &model.ProjectCdnConfig{
		NamespaceCode: "ns1",
		ProjectCode:   "proj1",
		Name:          "prod",
		Type:          "cloudflare",
		Enabled:       types.Ptr(true),
		Settings:      cdn.Values{cdn.FieldHosts: []string{"www.example.com"}},
	}
	assert.NoError(t, db.Create(config).Error)

	return db, config.ID
}

func newTestPurgeJob(configID int64, status model.CdnPurgeStatus) *model.ProjectCdnPurgeJob {
	return &model.ProjectCdnPurgeJob{
		NamespaceCode:  "ns1",
		ProjectCode:    "proj1",
		CdnConfigID:    configID,
		CdnConfigName:  "prod",
		CdnConfigType:  "cloudflare",
		ProjectVersion: 1,
		Status:         status,
		Targets:        []cdn.Target{{Kind: cdn.KindURL, Host: "www.example.com", Path: "/promo"}},
		Requested:      1,
	}
}

func TestProjectCdnPurgeJobRepository_Create(t *testing.T) {
	t.Run("successRoundTripsTargets", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		job := newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)

		assert.NoError(t, repo.Create(context.Background(), job))

		got, err := repo.FindByID(context.Background(), job.ID)

		assert.NoError(t, err)
		assert.Equal(t, []cdn.Target{{Kind: cdn.KindURL, Host: "www.example.com", Path: "/promo"}}, got.Targets)
	})
}

func TestProjectCdnPurgeJobRepository_FindMergeable(t *testing.T) {
	t.Run("successWithAWaitingJob", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)))

		got, err := repo.FindMergeable(context.Background(), nil, configID)

		assert.NoError(t, err)
		assert.Equal(t, model.CdnPurgeStatusWaitingAgents, got.Status)
	})

	t.Run("failWithARunningJob", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestPurgeJob(configID, model.CdnPurgeStatusRunning)))

		_, err := repo.FindMergeable(context.Background(), nil, configID)

		// A running job has batches on their way, it is too late to add to it.
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("failWithAnotherConfiguration", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)))

		_, err := repo.FindMergeable(context.Background(), nil, configID+999)

		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestProjectCdnPurgeJobRepository_FindActive(t *testing.T) {
	t.Run("successKeepsOnlyTheUnfinishedOnes", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		ctx := context.Background()

		assert.NoError(t, repo.Create(ctx, newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)))
		assert.NoError(t, repo.Create(ctx, newTestPurgeJob(configID, model.CdnPurgeStatusRunning)))
		assert.NoError(t, repo.Create(ctx, newTestPurgeJob(configID, model.CdnPurgeStatusSucceeded)))
		assert.NoError(t, repo.Create(ctx, newTestPurgeJob(configID, model.CdnPurgeStatusSkipped)))

		got, err := repo.FindActive(ctx, 0)

		assert.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("successRespectsTheLimitPerStatus", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		ctx := context.Background()

		assert.NoError(t, repo.Create(ctx, newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)))
		assert.NoError(t, repo.Create(ctx, newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)))
		assert.NoError(t, repo.Create(ctx, newTestPurgeJob(configID, model.CdnPurgeStatusRunning)))
		assert.NoError(t, repo.Create(ctx, newTestPurgeJob(configID, model.CdnPurgeStatusRunning)))

		got, err := repo.FindActive(ctx, 1)

		assert.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, model.CdnPurgeStatusRunning, got[0].Status)
		assert.Equal(t, model.CdnPurgeStatusWaitingAgents, got[1].Status)
	})

	t.Run("successPutsRunningJobsBeforeOlderWaitingOnes", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		ctx := context.Background()

		waiting := newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)
		assert.NoError(t, repo.Create(ctx, waiting))
		running := newTestPurgeJob(configID, model.CdnPurgeStatusRunning)
		assert.NoError(t, repo.Create(ctx, running))

		got, err := repo.FindActive(ctx, 1)

		// The older waiting job no longer takes the place of a job ready to send.
		assert.NoError(t, err)
		assert.Equal(t, []int64{running.ID, waiting.ID}, []int64{got[0].ID, got[1].ID})
	})
}

func TestProjectCdnPurgeJobRepository_MergeWaiting(t *testing.T) {
	merged := func(job *model.ProjectCdnPurgeJob) {
		job.Targets = append(job.Targets, cdn.Target{Kind: cdn.KindURL, Host: "www.example.com", Path: "/sale"})
		job.Requested = 2
		job.Skipped = map[string]int{"PATTERN": 1}
		job.ProjectVersion = 2
	}

	tests := []struct {
		name       string
		status     model.CdnPurgeStatus
		readAt     int
		wantSaved  bool
		wantStatus model.CdnPurgeStatus
	}{
		{name: "successWithAJobStillWaiting", status: model.CdnPurgeStatusWaitingAgents, readAt: 1, wantSaved: true},
		{name: "failWhenTheWorkerStartedTheJob", status: model.CdnPurgeStatusRunning, readAt: 1},
		{name: "failWhenAnotherPublishMergedFirst", status: model.CdnPurgeStatusWaitingAgents, readAt: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, configID := setupProjectCdnPurgeJobTestDB(t)
			repo := NewProjectCdnPurgeJobRepository(db)
			ctx := context.Background()
			job := newTestPurgeJob(configID, tt.status)
			assert.NoError(t, repo.Create(ctx, job))

			merged(job)
			saved, err := repo.MergeWaiting(ctx, nil, job, tt.readAt)

			assert.NoError(t, err)
			assert.Equal(t, tt.wantSaved, saved)
			got, err := repo.FindByID(ctx, job.ID)
			assert.NoError(t, err)
			if tt.wantSaved {
				assert.Len(t, got.Targets, 2)
				assert.Equal(t, 2, got.Requested)
				assert.Equal(t, map[string]int{"PATTERN": 1}, got.Skipped)
				assert.Equal(t, 2, got.ProjectVersion)
			} else {
				assert.Len(t, got.Targets, 1)
				assert.Equal(t, 1, got.ProjectVersion)
			}
		})
	}
}

func TestProjectCdnPurgeJobRepository_MarkRunning(t *testing.T) {
	tests := []struct {
		name      string
		status    model.CdnPurgeStatus
		version   int
		wantMoved bool
	}{
		{name: "successWithTheVersionTheAgentsWereCheckedAgainst", status: model.CdnPurgeStatusWaitingAgents, version: 1, wantMoved: true},
		{name: "failWhenAPublishMergedSinceTheRead", status: model.CdnPurgeStatusWaitingAgents, version: 0},
		{name: "failWhenTheJobIsNoLongerWaiting", status: model.CdnPurgeStatusSkipped, version: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, configID := setupProjectCdnPurgeJobTestDB(t)
			repo := NewProjectCdnPurgeJobRepository(db)
			ctx := context.Background()
			stored := newTestPurgeJob(configID, tt.status)
			assert.NoError(t, repo.Create(ctx, stored))

			read := *stored
			read.ProjectVersion = tt.version
			startedAt := time.Now()

			moved, err := repo.MarkRunning(ctx, &read, startedAt, false)

			assert.NoError(t, err)
			assert.Equal(t, tt.wantMoved, moved)
			got, err := repo.FindByID(ctx, stored.ID)
			assert.NoError(t, err)
			if tt.wantMoved {
				assert.Equal(t, model.CdnPurgeStatusRunning, got.Status)
				assert.NotNil(t, got.StartedAt)
				assert.False(t, got.WaitedForAgents)
				assert.Equal(t, model.CdnPurgeStatusRunning, read.Status)
			} else {
				assert.Equal(t, tt.status, got.Status)
				assert.Nil(t, got.StartedAt)
				assert.Equal(t, tt.status, read.Status)
			}
		})
	}
}

func TestProjectCdnPurgeJobRepository_FindByProject(t *testing.T) {
	t.Run("successNewestFirst", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		ctx := context.Background()

		first := newTestPurgeJob(configID, model.CdnPurgeStatusSucceeded)
		assert.NoError(t, repo.Create(ctx, first))
		second := newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)
		assert.NoError(t, repo.Create(ctx, second))

		got, err := repo.FindByProject(ctx, "ns1", "proj1", 0)

		assert.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, second.ID, got[0].ID)
	})

	t.Run("successWithAnotherProject", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		assert.NoError(t, repo.Create(context.Background(), newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)))

		got, err := repo.FindByProject(context.Background(), "ns1", "other", 0)

		assert.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestProjectCdnPurgeJobRepository_DeleteFinishedBefore(t *testing.T) {
	t.Run("successKeepsTheUnfinishedAndTheRecent", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		ctx := context.Background()

		old := newTestPurgeJob(configID, model.CdnPurgeStatusSucceeded)
		old.FinishedAt = types.Ptr(time.Now().Add(-48 * time.Hour))
		assert.NoError(t, repo.Create(ctx, old))

		recent := newTestPurgeJob(configID, model.CdnPurgeStatusSucceeded)
		recent.FinishedAt = types.Ptr(time.Now())
		assert.NoError(t, repo.Create(ctx, recent))

		running := newTestPurgeJob(configID, model.CdnPurgeStatusRunning)
		assert.NoError(t, repo.Create(ctx, running))

		deleted, err := repo.DeleteFinishedBefore(ctx, time.Now().Add(-24*time.Hour), 100)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		var remaining int64
		assert.NoError(t, db.Model(&model.ProjectCdnPurgeJob{}).Count(&remaining).Error)
		assert.Equal(t, int64(2), remaining)
	})

	t.Run("successDeletesInBatches", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			job := newTestPurgeJob(configID, model.CdnPurgeStatusSucceeded)
			job.FinishedAt = types.Ptr(time.Now().Add(-48 * time.Hour))
			assert.NoError(t, repo.Create(ctx, job))
		}

		deleted, err := repo.DeleteFinishedBefore(ctx, time.Now().Add(-24*time.Hour), 2)

		assert.NoError(t, err)
		assert.Equal(t, int64(5), deleted)
	})

	t.Run("successWithNothingToDelete", func(t *testing.T) {
		db, _ := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)

		deleted, err := repo.DeleteFinishedBefore(context.Background(), time.Now(), 100)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), deleted)
	})
}

func TestProjectCdnPurgeJobRepository_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		ctx := context.Background()
		job := newTestPurgeJob(configID, model.CdnPurgeStatusWaitingAgents)
		assert.NoError(t, repo.Create(ctx, job))

		job.Status = model.CdnPurgeStatusSucceeded
		job.Targets = nil
		job.Purged = 1
		assert.NoError(t, repo.Update(ctx, job))

		got, err := repo.FindByID(ctx, job.ID)

		assert.NoError(t, err)
		assert.Equal(t, model.CdnPurgeStatusSucceeded, got.Status)
		assert.Nil(t, got.Targets)
		assert.Equal(t, 1, got.Purged)
	})
}

func TestProjectCdnPurgeJobRepository_UpdateProgress(t *testing.T) {
	t.Run("successSavesOnlyTheProgress", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		ctx := context.Background()
		job := newTestPurgeJob(configID, model.CdnPurgeStatusRunning)
		assert.NoError(t, repo.Create(ctx, job))

		job.Cursor = 1
		job.Purged = 1
		job.Failed = 2
		job.Attempts = 3
		job.Error = "rate limited"
		// Not part of the progress: must be left as stored.
		job.Status = model.CdnPurgeStatusSucceeded
		job.Targets = nil
		assert.NoError(t, repo.UpdateProgress(ctx, job))

		got, err := repo.FindByID(ctx, job.ID)

		assert.NoError(t, err)
		assert.Equal(t, 1, got.Cursor)
		assert.Equal(t, 1, got.Purged)
		assert.Equal(t, 2, got.Failed)
		assert.Equal(t, 3, got.Attempts)
		assert.Equal(t, "rate limited", got.Error)
		assert.Equal(t, model.CdnPurgeStatusRunning, got.Status)
		assert.Len(t, got.Targets, 1)
	})

	t.Run("failWithACancelledContext", func(t *testing.T) {
		db, configID := setupProjectCdnPurgeJobTestDB(t)
		repo := NewProjectCdnPurgeJobRepository(db)
		job := newTestPurgeJob(configID, model.CdnPurgeStatusRunning)
		assert.NoError(t, repo.Create(context.Background(), job))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// The reason the worker saves with a context of its own.
		assert.Error(t, repo.UpdateProgress(ctx, job))
	})
}

func TestProjectCdnPurgeJobRepository_GetQuery(t *testing.T) {
	db, _ := setupProjectCdnPurgeJobTestDB(t)
	repo := NewProjectCdnPurgeJobRepository(db)

	assert.NotNil(t, repo.GetTx(context.Background()))
	assert.NotNil(t, repo.GetQuery(context.Background()))
}
