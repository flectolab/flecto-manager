package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/flectolab/flecto-manager/cdn"
	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/crypt"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// workerProviderType is registered by these tests so a purge can be driven without
// a network call.
const workerProviderType = "worker-fake"

type workerProvider struct {
	// batches records what was actually sent, which is how batching and resuming
	// are checked.
	batches [][]cdn.Target
	// errs is consumed one per call, nil meaning success.
	errs []error
	// verifyErr is returned by Verify.
	verifyErr error
	limits    cdn.Limits
	// onPurge, when set, runs on every call with its 0-based index before errs is
	// consumed. It is how a test ends the run in the middle of a request.
	onPurge func(call int)
}

func (p *workerProvider) Type() string                   { return workerProviderType }
func (p *workerProvider) Capabilities() cdn.Capabilities { return cdn.Capabilities{URL: true} }
func (p *workerProvider) Limits() cdn.Limits             { return p.limits }
func (p *workerProvider) Verify(context.Context) error   { return p.verifyErr }

func (p *workerProvider) Purge(_ context.Context, batch []cdn.Target) error {
	p.batches = append(p.batches, batch)
	if p.onPurge != nil {
		p.onPurge(len(p.batches) - 1)
	}

	if len(p.errs) == 0 {
		return nil
	}
	err := p.errs[0]
	p.errs = p.errs[1:]
	return err
}

// registerWorkerProvider installs the fake and hands back the instance every
// build returns, so a test can read what it was sent.
func registerWorkerProvider(limits cdn.Limits, errs ...error) *workerProvider {
	provider := &workerProvider{limits: limits, errs: errs}

	cdn.Register(cdn.Descriptor{
		Type:   workerProviderType,
		Label:  "Worker fake",
		Fields: []cdn.Field{{Name: cdn.FieldHosts, Label: "Hosts", Kind: cdn.FieldKindStringList, Required: true}},
	}, func(settings, secrets cdn.Values, options cdn.Options) (cdn.Provider, error) {
		return provider, nil
	})

	return provider
}

type workerTestDeps struct {
	ctrl      *gomock.Controller
	appCtx    *appContext.Context
	agentRepo *mockFlectoRepository.MockAgentRepository
	db        *gorm.DB
	jobRepo   repository.ProjectCdnPurgeJobRepository
	worker    CdnPurgeWorker
	configID  int64
}

func setupCdnPurgeWorkerTest(t *testing.T) *workerTestDeps {
	t.Helper()

	ctrl := gomock.NewController(t)
	appCtx := appContext.TestContext(nil)

	key, err := crypt.GenerateKey()
	assert.NoError(t, err)
	cipher, err := crypt.NewAESGCM(key)
	assert.NoError(t, err)
	crypt.RegisterSerializer(cipher)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(
		&model.Namespace{}, &model.Project{}, &model.ProjectCdnConfig{},
		&model.ProjectCdnPurgeJob{}, &model.ActivityEvent{},
	))
	assert.NoError(t, db.Create(&model.Namespace{NamespaceCode: "ns1", Name: "ns1"}).Error)
	assert.NoError(t, db.Create(&model.Project{NamespaceCode: "ns1", ProjectCode: "proj1", Name: "proj1"}).Error)

	config := &model.ProjectCdnConfig{
		NamespaceCode: "ns1",
		ProjectCode:   "proj1",
		Name:          "prod",
		Type:          workerProviderType,
		Enabled:       types.Ptr(true),
		Settings:      cdn.Values{cdn.FieldHosts: []string{"www.example.com"}},
	}
	assert.NoError(t, db.Create(config).Error)

	agentRepo := mockFlectoRepository.NewMockAgentRepository(ctrl)
	jobRepo := repository.NewProjectCdnPurgeJobRepository(db)

	return &workerTestDeps{
		ctrl:      ctrl,
		appCtx:    appCtx,
		agentRepo: agentRepo,
		db:        db,
		jobRepo:   jobRepo,
		configID:  config.ID,
		worker: NewCdnPurgeWorker(
			appCtx,
			jobRepo,
			repository.NewProjectCdnConfigRepository(db),
			agentRepo,
			NewActivityService(appCtx, repository.NewActivityEventRepository(db)),
		),
	}
}

func (d *workerTestDeps) queueJob(t *testing.T, targetCount int) *model.ProjectCdnPurgeJob {
	t.Helper()

	targets := make([]cdn.Target, 0, targetCount)
	for i := 0; i < targetCount; i++ {
		targets = append(targets, cdn.Target{Kind: cdn.KindURL, Host: "www.example.com", Path: pathOf(i)})
	}

	job := &model.ProjectCdnPurgeJob{
		NamespaceCode:   "ns1",
		ProjectCode:     "proj1",
		CdnConfigID:     d.configID,
		CdnConfigName:   "prod",
		CdnConfigType:   workerProviderType,
		ProjectVersion:  2,
		Status:          model.CdnPurgeStatusWaitingAgents,
		Targets:         targets,
		Requested:       targetCount,
		WaitedForAgents: true,
		DeadlineAt:      time.Now().Add(time.Hour),
	}
	assert.NoError(t, d.jobRepo.Create(context.Background(), job))
	return job
}

func pathOf(i int) string {
	return "/p" + string(rune('a'+i%26)) + string(rune('0'+i/26))
}

// convergedAgent is an agent already serving the version the job waits for.
func convergedAgent(version int) model.Agent {
	return model.Agent{
		NamespaceCode: "ns1",
		ProjectCode:   "proj1",
		Agent:         commonTypes.Agent{Name: "agent-1", Version: version},
		LastHitAt:     time.Now(),
	}
}

func (d *workerTestDeps) reload(t *testing.T, id int64) *model.ProjectCdnPurgeJob {
	t.Helper()

	job, err := d.jobRepo.FindByID(context.Background(), id)
	assert.NoError(t, err)
	return job
}

func TestCdnPurgeWorker_Run(t *testing.T) {
	t.Run("successPurgesAConvergedJob", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 3)
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{convergedAgent(2)}, nil)

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 1, finished)

		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusSucceeded, got.Status)
		assert.Equal(t, 3, got.Purged)
		assert.Len(t, provider.batches, 1)
		// The addresses of a purge that went through are of no use to anyone.
		assert.Nil(t, got.Targets)
		assert.NotNil(t, got.FinishedAt)
	})

	t.Run("successSplitsIntoBatchesOfTheProviderSize", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 2, TargetsPerSecond: 1000})
		deps.queueJob(t, 5)
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{convergedAgent(2)}, nil)

		_, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Len(t, provider.batches, 3)
		assert.Len(t, provider.batches[0], 2)
		assert.Len(t, provider.batches[2], 1)
	})

	t.Run("successRecordsTheActivity", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 1000})
		deps.queueJob(t, 1)
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{convergedAgent(2)}, nil)

		_, err := deps.worker.Run(context.Background())
		assert.NoError(t, err)

		var events []model.ActivityEvent
		assert.NoError(t, deps.db.Find(&events).Error)
		assert.Len(t, events, 1)
		assert.Equal(t, model.ActivityResourceCdn, events[0].Resource)
		assert.Equal(t, model.ActivityActionPurge, events[0].Action)
		// No user context behind a background task: the system is what acted.
		assert.Equal(t, model.ActivityActorSystem, events[0].Actor)
	})

	t.Run("successHoldsBackWhileAgentsAreBehind", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 1)
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{convergedAgent(1)}, nil)

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 0, finished)
		// Purging now would let the CDN cache the old response again, from an agent
		// that has not caught up.
		assert.Empty(t, provider.batches)
		assert.Equal(t, model.CdnPurgeStatusWaitingAgents, deps.reload(t, job.ID).Status)
	})

	t.Run("successPurgesPastTheDeadlineEvenIfAgentsAreBehind", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 1)
		job.DeadlineAt = time.Now().Add(-time.Minute)
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{convergedAgent(1)}, nil)

		_, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusSucceeded, got.Status)
		// Flagged, because the cache may hold the previous version again.
		assert.False(t, got.WaitedForAgents)
	})

	t.Run("successPurgesWhenNoAgentIsOnline", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 1)

		offline := convergedAgent(1)
		offline.LastHitAt = time.Now().Add(-24 * time.Hour)
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{offline}, nil)

		_, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		// Nothing is going to serve the old response back, so there is nothing to
		// wait for.
		assert.Equal(t, model.CdnPurgeStatusSucceeded, deps.reload(t, job.ID).Status)
	})

	t.Run("successSkipsAJobWhoseConfigurationWasDisabled", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 1)
		assert.NoError(t, deps.db.Model(&model.ProjectCdnConfig{}).
			Where("id = ?", deps.configID).Update("enabled", false).Error)

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 1, finished)
		assert.Empty(t, provider.batches)
		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusSkipped, got.Status)
		assert.Equal(t, model.CdnPurgeSkipReasonConfigDisabled, got.SkipReason)
	})

	t.Run("successDefersOnATemporaryFailure", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		registerWorkerProvider(
			cdn.Limits{MaxTargetsPerRequest: 1, TargetsPerSecond: 1000},
			nil, cdn.Temporary(errors.New("rate limited"), 0),
		)
		job := deps.queueJob(t, 3)
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{convergedAgent(2)}, nil)

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 0, finished)

		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusRunning, got.Status)
		assert.Equal(t, 1, got.Attempts)
		// The first batch went through, so the next run resumes after it.
		assert.Equal(t, 1, got.Cursor)
		assert.Equal(t, 1, got.Purged)
		assert.Contains(t, got.Error, "rate limited")
	})

	t.Run("successResetsTheAttemptsAfterABatchGoesThrough", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 1, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 2)
		job.Status = model.CdnPurgeStatusRunning
		// One failure short of giving up, left by earlier runs.
		job.Attempts = cdnMaxAttempts - 1
		job.Error = "rate limited"
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))
		// The first batch goes through, the second meets a new rate limit.
		provider.errs = []error{nil, cdn.Temporary(errors.New("rate limited again"), 0)}

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 0, finished)

		got := deps.reload(t, job.ID)
		// A new streak starts: the job is deferred, not given up on.
		assert.Equal(t, model.CdnPurgeStatusRunning, got.Status)
		assert.Equal(t, 1, got.Attempts)
		assert.Equal(t, 1, got.Purged)
		assert.Equal(t, 0, got.Failed)
	})

	t.Run("successResumesWhereItStopped", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 1, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 3)
		job.Status = model.CdnPurgeStatusRunning
		job.Cursor = 2
		job.Purged = 2
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))

		_, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		// A running job is past the convergence check, and only the tail is resent.
		assert.Len(t, provider.batches, 1)
		assert.Equal(t, 3, deps.reload(t, job.ID).Purged)
	})

	t.Run("successFailsForGoodOnAPermanentError", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(
			cdn.Limits{MaxTargetsPerRequest: 1, TargetsPerSecond: 1000},
			errors.New("authentication error"),
		)
		job := deps.queueJob(t, 3)
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{convergedAgent(2)}, nil)

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 1, finished)

		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusFailed, got.Status)
		assert.Equal(t, 3, got.Failed)
		assert.Contains(t, got.Error, "authentication error")
		// A bad credential does not fix itself between two batches, so the rest is
		// not sent one by one to fail the same way.
		assert.Len(t, provider.batches, 1)
	})

	t.Run("successReportsPartialWhenSomeBatchesWentThrough", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		registerWorkerProvider(
			cdn.Limits{MaxTargetsPerRequest: 1, TargetsPerSecond: 1000},
			nil, errors.New("boom"),
		)
		job := deps.queueJob(t, 3)
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{convergedAgent(2)}, nil)

		_, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusPartial, got.Status)
		assert.Equal(t, 1, got.Purged)
		assert.Equal(t, 2, got.Failed)
		// A partial job keeps its targets, so what was missed stays knowable.
		assert.NotNil(t, got.Targets)
	})

	t.Run("successGivesUpAfterTooManyTemporaryFailures", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		registerWorkerProvider(
			cdn.Limits{MaxTargetsPerRequest: 1, TargetsPerSecond: 1000},
			cdn.Temporary(errors.New("still down"), 0),
		)
		job := deps.queueJob(t, 1)
		job.Status = model.CdnPurgeStatusRunning
		job.Attempts = cdnMaxAttempts - 1
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))

		_, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusFailed, got.Status)
		assert.Equal(t, cdnMaxAttempts, got.Attempts)
	})

	t.Run("successFailsWhenTheProviderCannotBeBuilt", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		job := deps.queueJob(t, 1)
		assert.NoError(t, deps.db.Model(&model.ProjectCdnConfig{}).
			Where("id = ?", deps.configID).Update("type", "gone-from-this-build").Error)
		assert.NoError(t, deps.db.Model(&model.ProjectCdnPurgeJob{}).
			Where("id = ?", job.ID).Update("status", model.CdnPurgeStatusRunning).Error)

		_, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusFailed, got.Status)
		assert.Contains(t, got.Error, "unknown cdn provider type")
	})

	t.Run("successSavesTheProgressWhenTheRunIsCutShort", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 1, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 3)
		job.Status = model.CdnPurgeStatusRunning
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// The run ends while the second batch is in flight, which a real client
		// reports as a transport error, so a temporary one.
		provider.onPurge = func(call int) {
			if call == 1 {
				cancel()
				provider.errs = []error{cdn.Temporary(context.Canceled, 0)}
			}
		}

		finished, err := deps.worker.Run(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 0, finished)

		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusRunning, got.Status)
		// The first batch is on record, the interrupted one is not.
		assert.Equal(t, 1, got.Cursor)
		assert.Equal(t, 1, got.Purged)
		// Being cut short is not the provider's fault: no attempt spent, no error.
		assert.Equal(t, 0, got.Attempts)
		assert.Empty(t, got.Error)

		// The next run resends the interrupted batch and the rest, nothing before.
		provider.onPurge = nil
		provider.batches = nil

		finished, err = deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 1, finished)
		assert.Len(t, provider.batches, 2)

		got = deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusSucceeded, got.Status)
		assert.Equal(t, 3, got.Purged)
	})

	t.Run("successFinishesOnTheLastInstantOfTheRun", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 1, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 2)
		job.Status = model.CdnPurgeStatusRunning
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// The last batch goes through, and the run ends right after it.
		provider.onPurge = func(call int) {
			if call == 1 {
				cancel()
			}
		}

		finished, err := deps.worker.Run(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 1, finished)

		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusSucceeded, got.Status)
		assert.Equal(t, 2, got.Purged)

		var events int64
		assert.NoError(t, deps.db.Model(&model.ActivityEvent{}).Count(&events).Error)
		assert.Equal(t, int64(1), events)
	})

	t.Run("successSendsAManualPurgeWithoutLookingAtTheAgents", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 10, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 2)
		job.ProjectVersion = 0
		job.Status = model.CdnPurgeStatusRunning
		job.StartedAt = types.Ptr(time.Now())
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))
		// No FindByProject expectation: gomock fails the test if the agents are read.

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 1, finished)
		assert.Len(t, provider.batches, 1)
		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusSucceeded, got.Status)
		assert.True(t, got.WaitedForAgents)
	})

	t.Run("successStopsWhenTheRunIsOutOfTime", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 1, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 3)
		job.Status = model.CdnPurgeStatusRunning
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		finished, err := deps.worker.Run(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 0, finished)
		assert.Empty(t, provider.batches)
	})

	t.Run("successWhenDisabled", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		deps.appCtx.Config.Cdn.Enabled = false
		deps.queueJob(t, 1)

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 0, finished)
	})

	t.Run("successWithoutJob", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 0, finished)
	})

	t.Run("successLeavesAJobAPublishMergedIntoWhileItWasRead", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		provider := registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 1000})
		job := deps.queueJob(t, 1)

		// A publish folds version 3 into the job while the worker checks the agents
		// against version 2.
		deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").
			DoAndReturn(func(context.Context, string, string) ([]model.Agent, error) {
				assert.NoError(t, deps.db.Model(&model.ProjectCdnPurgeJob{}).Where("id = ?", job.ID).
					Updates(map[string]any{"project_version": 3, "requested": 2}).Error)
				return []model.Agent{convergedAgent(2)}, nil
			})

		finished, err := deps.worker.Run(context.Background())

		// Nothing is sent from the stale copy: the next run checks the agents
		// against version 3 and sends the merged targets.
		assert.NoError(t, err)
		assert.Equal(t, 0, finished)
		assert.Empty(t, provider.batches)
		got := deps.reload(t, job.ID)
		assert.Equal(t, model.CdnPurgeStatusWaitingAgents, got.Status)
		assert.Equal(t, 3, got.ProjectVersion)
		assert.Nil(t, got.StartedAt)
	})

	t.Run("successCarriesOnAfterOneJobFails", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		registerWorkerProvider(cdn.Limits{MaxTargetsPerRequest: 100, TargetsPerSecond: 1000})
		deps.queueJob(t, 1)
		deps.queueJob(t, 1)

		// The agent lookup fails for the first job and succeeds for the second.
		gomock.InOrder(
			deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return(nil, errors.New("boom")),
			deps.agentRepo.EXPECT().FindByProject(gomock.Any(), "ns1", "proj1").Return([]model.Agent{convergedAgent(2)}, nil),
		)

		finished, err := deps.worker.Run(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 1, finished)
	})
}

func TestTruncateError(t *testing.T) {
	ascii := strings.Repeat("a", cdnErrorMaxLength)

	tests := []struct {
		name    string
		message string
		want    string
	}{
		{name: "successWithAShortMessage", message: "boom", want: "boom"},
		{name: "successCutsALongMessage", message: ascii + "b", want: ascii},
		{
			// "é" is two bytes; the limit falls between them.
			name:    "successCutsBeforeACharacterSplitByTheLimit",
			message: strings.Repeat("a", cdnErrorMaxLength-1) + "é",
			want:    strings.Repeat("a", cdnErrorMaxLength-1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateError(tt.message)

			assert.Equal(t, tt.want, got)
			assert.True(t, utf8.ValidString(got))
		})
	}
}

func TestCdnPurgeWorker_PurgeExpiredJobs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		deps.appCtx.Config.Cdn.JobRetention = time.Hour

		job := deps.queueJob(t, 1)
		job.Status = model.CdnPurgeStatusSucceeded
		job.FinishedAt = types.Ptr(time.Now().Add(-2 * time.Hour))
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))

		purged, err := deps.worker.PurgeExpiredJobs(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, int64(1), purged)
	})

	t.Run("successWithoutRetention", func(t *testing.T) {
		deps := setupCdnPurgeWorkerTest(t)
		defer deps.ctrl.Finish()
		deps.appCtx.Config.Cdn.JobRetention = 0

		job := deps.queueJob(t, 1)
		job.Status = model.CdnPurgeStatusSucceeded
		job.FinishedAt = types.Ptr(time.Now().Add(-10000 * time.Hour))
		assert.NoError(t, deps.jobRepo.Update(context.Background(), job))

		purged, err := deps.worker.PurgeExpiredJobs(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, int64(0), purged)
	})
}
