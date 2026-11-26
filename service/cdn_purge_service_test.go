package service

import (
	"context"
	"errors"
	"testing"
	"time"

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

// newTestCdnPurgeService returns a purge service that finds no configuration.
func newTestCdnPurgeService(t *testing.T) CdnPurgeService {
	t.Helper()

	ctrl := gomock.NewController(t)
	configRepo := mockFlectoRepository.NewMockProjectCdnConfigRepository(ctrl)
	configRepo.EXPECT().
		FindEnabledByProject(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil).
		AnyTimes()

	return NewCdnPurgeService(appContext.TestContext(nil), configRepo, nil, nil, nil, nil)
}

type cdnPurgeTestDeps struct {
	ctrl         *gomock.Controller
	appCtx       *appContext.Context
	configRepo   *mockFlectoRepository.MockProjectCdnConfigRepository
	redirectRepo *mockFlectoRepository.MockRedirectRepository
	pageRepo     *mockFlectoRepository.MockPageRepository
	db           *gorm.DB
	svc          CdnPurgeService
}

// setupCdnPurgeTest uses a real database for writes, which go through the publish transaction.
func setupCdnPurgeTest(t *testing.T) *cdnPurgeTestDeps {
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

	configRepo := mockFlectoRepository.NewMockProjectCdnConfigRepository(ctrl)
	redirectRepo := mockFlectoRepository.NewMockRedirectRepository(ctrl)
	pageRepo := mockFlectoRepository.NewMockPageRepository(ctrl)

	return &cdnPurgeTestDeps{
		ctrl:         ctrl,
		appCtx:       appCtx,
		configRepo:   configRepo,
		redirectRepo: redirectRepo,
		pageRepo:     pageRepo,
		db:           db,
		svc: NewCdnPurgeService(
			appCtx, configRepo, repository.NewProjectCdnPurgeJobRepository(db), redirectRepo, pageRepo,
			NewActivityService(appCtx, repository.NewActivityEventRepository(db)),
		),
	}
}

func testCdnConfig(id int64, name string, hosts []string) model.ProjectCdnConfig {
	return model.ProjectCdnConfig{
		ID:            id,
		NamespaceCode: "ns1",
		ProjectCode:   "proj1",
		Name:          name,
		Type:          "cloudflare",
		Enabled:       types.Ptr(true),
		Settings:      cdn.Values{cdn.FieldHosts: hosts},
	}
}

func TestCdnPurgeService_Collect(t *testing.T) {
	t.Run("successGathersNewAndOldAddresses", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()

		deps.configRepo.EXPECT().FindEnabledByProject(ctx, "ns1", "proj1").
			Return([]model.ProjectCdnConfig{testCdnConfig(1, "prod", []string{"www.example.com"})}, nil)
		deps.redirectRepo.EXPECT().FindAddressesByIDs(ctx, nil, []int64{7, 8}, cdnAddressBatchSize).
			Return([]model.ResourceAddress{
				{ID: 7, Type: "BASIC", Source: "/old"},
				{ID: 8, Type: "BASIC", Source: "/old-ignored", Options: commonTypes.RedirectOptions{IgnoreQueryString: types.Ptr(true)}},
			}, nil)
		deps.pageRepo.EXPECT().FindAddressesByIDs(ctx, nil, []int64(nil), cdnAddressBatchSize).
			Return(nil, nil)

		plan, err := deps.svc.Collect(ctx, "ns1", "proj1", model.CdnDraftChanges{
			RedirectIDs: []int64{7, 8},
			NewEntries:  []cdn.Entry{{Type: "BASIC", Source: "/new"}},
		})

		assert.NoError(t, err)
		assert.Equal(t, []cdn.Entry{
			{Type: "BASIC", Source: "/new"},
			{Type: "BASIC", Source: "/old"},
			{Type: "BASIC", Source: "/old-ignored", IgnoreQueryString: true},
		}, plan.Entries)
		assert.Len(t, plan.Configs, 1)
	})

	t.Run("successWithoutConfigurationSkipsTheAddressReads", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()

		// One query only: a project with no CDN must not pay for the feature.
		deps.configRepo.EXPECT().FindEnabledByProject(ctx, "ns1", "proj1").Return(nil, nil)

		plan, err := deps.svc.Collect(ctx, "ns1", "proj1", model.CdnDraftChanges{RedirectIDs: []int64{7}})

		assert.NoError(t, err)
		assert.True(t, plan.IsEmpty())
	})

	t.Run("successWhenDisabledGlobally", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		deps.appCtx.Config.Cdn.Enabled = false

		plan, err := deps.svc.Collect(context.Background(), "ns1", "proj1", model.CdnDraftChanges{RedirectIDs: []int64{7}})

		assert.NoError(t, err)
		assert.True(t, plan.IsEmpty())
	})

	t.Run("failWhenConfigLookupFails", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()

		deps.configRepo.EXPECT().FindEnabledByProject(ctx, "ns1", "proj1").Return(nil, errors.New("boom"))

		_, err := deps.svc.Collect(ctx, "ns1", "proj1", model.CdnDraftChanges{})

		assert.ErrorContains(t, err, "boom")
	})

	t.Run("failWhenRedirectAddressesFail", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()

		deps.configRepo.EXPECT().FindEnabledByProject(ctx, "ns1", "proj1").
			Return([]model.ProjectCdnConfig{testCdnConfig(1, "prod", []string{"www.example.com"})}, nil)
		deps.redirectRepo.EXPECT().FindAddressesByIDs(ctx, nil, []int64{7}, cdnAddressBatchSize).
			Return(nil, errors.New("boom"))

		_, err := deps.svc.Collect(ctx, "ns1", "proj1", model.CdnDraftChanges{RedirectIDs: []int64{7}})

		assert.ErrorContains(t, err, "boom")
	})

	t.Run("failWhenPageAddressesFail", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()

		deps.configRepo.EXPECT().FindEnabledByProject(ctx, "ns1", "proj1").
			Return([]model.ProjectCdnConfig{testCdnConfig(1, "prod", []string{"www.example.com"})}, nil)
		deps.redirectRepo.EXPECT().FindAddressesByIDs(ctx, nil, []int64(nil), cdnAddressBatchSize).Return(nil, nil)
		deps.pageRepo.EXPECT().FindAddressesByIDs(ctx, nil, []int64{3}, cdnAddressBatchSize).
			Return(nil, errors.New("boom"))

		_, err := deps.svc.Collect(ctx, "ns1", "proj1", model.CdnDraftChanges{PageIDs: []int64{3}})

		assert.ErrorContains(t, err, "boom")
	})
}

func TestCdnPurgeService_Enqueue(t *testing.T) {
	basePlan := func(entries []cdn.Entry, configs ...model.ProjectCdnConfig) model.CdnPublishPlan {
		return model.CdnPublishPlan{
			NamespaceCode: "ns1",
			ProjectCode:   "proj1",
			Entries:       entries,
			Configs:       configs,
		}
	}

	t.Run("successCreatesOneJobPerConfiguration", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()

		plan := basePlan(
			[]cdn.Entry{{Type: "BASIC", Source: "/promo"}},
			testCdnConfig(1, "prod", []string{"www.example.com"}),
			testCdnConfig(2, "eu", []string{"eu.example.com"}),
		)

		err := deps.svc.Enqueue(context.Background(), deps.db, plan, 4)

		assert.NoError(t, err)
		var jobs []model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.Order("cdn_config_id").Find(&jobs).Error)
		assert.Len(t, jobs, 2)
		assert.Equal(t, "prod", jobs[0].CdnConfigName)
		assert.Equal(t, "eu", jobs[1].CdnConfigName)
		assert.Equal(t, model.CdnPurgeStatusWaitingAgents, jobs[0].Status)
		assert.Equal(t, 4, jobs[0].ProjectVersion)
	})

	t.Run("successExpandsHostLessEntryOverEveryHost", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()

		plan := basePlan(
			[]cdn.Entry{{Type: "BASIC", Source: "/promo"}},
			testCdnConfig(1, "prod", []string{"www.example.com", "example.com"}),
		)

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, plan, 1))

		var job model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.First(&job).Error)
		assert.Equal(t, 2, job.Requested)
		assert.Equal(t, []cdn.Target{
			{Kind: cdn.KindURL, Host: "example.com", Path: "/promo"},
			{Kind: cdn.KindURL, Host: "www.example.com", Path: "/promo"},
		}, job.Targets)
	})

	t.Run("successPurgesByPrefixAnEntryIgnoringQueryString", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()

		plan := basePlan(
			[]cdn.Entry{
				{Type: "BASIC", Source: "/promo", IgnoreQueryString: true},
				{Type: "BASIC", Source: "/other"},
			},
			testCdnConfig(1, "prod", []string{"www.example.com"}),
		)

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, plan, 1))

		var job model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.First(&job).Error)
		assert.Equal(t, []cdn.Target{
			{Kind: cdn.KindPrefix, Host: "www.example.com", Path: "/promo"},
			{Kind: cdn.KindURL, Host: "www.example.com", Path: "/other"},
		}, job.Targets)
		assert.Empty(t, job.Skipped)
	})

	t.Run("successPurgesByUrlWhenTheProviderTypeIsUnknown", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()

		config := testCdnConfig(1, "prod", []string{"www.example.com"})
		config.Type = "removed-provider"
		plan := basePlan([]cdn.Entry{{Type: "BASIC", Source: "/promo", IgnoreQueryString: true}}, config)

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, plan, 1))

		var job model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.First(&job).Error)
		assert.Equal(t, []cdn.Target{{Kind: cdn.KindURL, Host: "www.example.com", Path: "/promo"}}, job.Targets)
		assert.Equal(t, map[string]int{string(cdn.SkipReasonQueryVariants): 1}, job.Skipped)
	})

	t.Run("successRecordsSkippedEntries", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()

		plan := basePlan(
			[]cdn.Entry{
				{Type: "BASIC", Source: "/promo"},
				{Type: "REGEX", Source: "^/a/(.*)$"},
				{Type: "BASIC_HOST", Source: "other.com/x"},
			},
			testCdnConfig(1, "prod", []string{"www.example.com"}),
		)

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, plan, 1))

		var job model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.First(&job).Error)
		assert.Equal(t, 1, job.Requested)
		assert.Equal(t, map[string]int{
			string(cdn.SkipReasonPattern):     1,
			string(cdn.SkipReasonNoHostMatch): 1,
		}, job.Skipped)
	})

	t.Run("successMergesIntoAJobStillWaiting", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		config := testCdnConfig(1, "prod", []string{"www.example.com"})

		assert.NoError(t, deps.svc.Enqueue(ctx, deps.db, basePlan([]cdn.Entry{{Type: "BASIC", Source: "/a"}}, config), 1))
		assert.NoError(t, deps.svc.Enqueue(ctx, deps.db, basePlan([]cdn.Entry{{Type: "BASIC", Source: "/b"}}, config), 2))

		var jobs []model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.Find(&jobs).Error)

		assert.Len(t, jobs, 1)
		assert.Equal(t, 2, jobs[0].Requested)
		assert.Equal(t, 2, jobs[0].ProjectVersion)
	})

	t.Run("successDeduplicatesOnMerge", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		config := testCdnConfig(1, "prod", []string{"www.example.com"})
		entries := []cdn.Entry{{Type: "BASIC", Source: "/a"}}

		assert.NoError(t, deps.svc.Enqueue(ctx, deps.db, basePlan(entries, config), 1))
		assert.NoError(t, deps.svc.Enqueue(ctx, deps.db, basePlan(entries, config), 2))

		var job model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.First(&job).Error)
		assert.Equal(t, 1, job.Requested)
	})

	t.Run("successCreatesANewJobWhenTheFirstIsRunning", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		config := testCdnConfig(1, "prod", []string{"www.example.com"})

		assert.NoError(t, deps.svc.Enqueue(ctx, deps.db, basePlan([]cdn.Entry{{Type: "BASIC", Source: "/a"}}, config), 1))
		assert.NoError(t, deps.db.Model(&model.ProjectCdnPurgeJob{}).
			Where("1 = 1").Update("status", model.CdnPurgeStatusRunning).Error)

		assert.NoError(t, deps.svc.Enqueue(ctx, deps.db, basePlan([]cdn.Entry{{Type: "BASIC", Source: "/b"}}, config), 2))

		var jobs []model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.Find(&jobs).Error)
		assert.Len(t, jobs, 2)
	})

	t.Run("successSkipsAJobOverTheCap", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		deps.appCtx.Config.Cdn.MaxTargetsPerJob = 2

		plan := basePlan(
			[]cdn.Entry{
				{Type: "BASIC", Source: "/a"},
				{Type: "BASIC", Source: "/b"},
				{Type: "BASIC", Source: "/c"},
			},
			testCdnConfig(1, "prod", []string{"www.example.com"}),
		)

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, plan, 1))

		var job model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.First(&job).Error)
		assert.Equal(t, model.CdnPurgeStatusSkipped, job.Status)
		assert.Equal(t, model.CdnPurgeSkipReasonTooManyTargets, job.SkipReason)
		assert.Nil(t, job.Targets)
		assert.NotNil(t, job.FinishedAt)

		var events []model.ActivityEvent
		assert.NoError(t, deps.db.Find(&events).Error)
		assert.Len(t, events, 1)
		assert.Equal(t, model.ActivityResourceCdn, events[0].Resource)
		assert.Equal(t, model.ActivityActionPurge, events[0].Action)
		assert.Equal(t, job.ID, *events[0].ResourceID)
	})

	t.Run("successJournalsNothingForAQueuedJob", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()

		plan := basePlan([]cdn.Entry{{Type: "BASIC", Source: "/a"}}, testCdnConfig(1, "prod", []string{"www.example.com"}))

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, plan, 1))

		var count int64
		assert.NoError(t, deps.db.Model(&model.ActivityEvent{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
	})

	t.Run("successKeepsTheWaitingJobWhenTheUnionExceedsTheCap", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		deps.appCtx.Config.Cdn.MaxTargetsPerJob = 2
		config := testCdnConfig(1, "prod", []string{"www.example.com"})

		assert.NoError(t, deps.svc.Enqueue(ctx, deps.db,
			basePlan([]cdn.Entry{{Type: "BASIC", Source: "/a"}, {Type: "BASIC", Source: "/b"}}, config), 1))
		assert.NoError(t, deps.svc.Enqueue(ctx, deps.db,
			basePlan([]cdn.Entry{{Type: "BASIC", Source: "/c"}}, config), 2))

		var jobs []model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.Order("id").Find(&jobs).Error)

		assert.Len(t, jobs, 2)
		assert.Equal(t, model.CdnPurgeStatusWaitingAgents, jobs[0].Status)
		assert.Equal(t, 2, jobs[0].Requested)
		assert.Equal(t, 1, jobs[0].ProjectVersion)
		assert.Equal(t, model.CdnPurgeStatusWaitingAgents, jobs[1].Status)
		assert.Equal(t, 1, jobs[1].Requested)
		assert.Equal(t, 2, jobs[1].ProjectVersion)
	})

	t.Run("successQueuesApartWhenTheWaitingJobChangedSinceItWasRead", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		config := testCdnConfig(1, "prod", []string{"www.example.com"})
		jobRepo := mockFlectoRepository.NewMockProjectCdnPurgeJobRepository(deps.ctrl)
		svc := NewCdnPurgeService(deps.appCtx, deps.configRepo, jobRepo, deps.redirectRepo, deps.pageRepo, nil)

		// A worker started the job between the read and the merge.
		read := &model.ProjectCdnPurgeJob{ID: 5, CdnConfigID: 1, ProjectVersion: 1, Status: model.CdnPurgeStatusWaitingAgents}
		jobRepo.EXPECT().FindMergeable(ctx, deps.db, int64(1)).Return(read, nil)
		jobRepo.EXPECT().MergeWaiting(ctx, deps.db, read, 1).Return(false, nil)

		assert.NoError(t, svc.Enqueue(ctx, deps.db, basePlan([]cdn.Entry{{Type: "BASIC", Source: "/b"}}, config), 2))

		var jobs []model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.Find(&jobs).Error)
		assert.Len(t, jobs, 1)
		assert.Equal(t, 2, jobs[0].ProjectVersion)
		assert.Equal(t, model.CdnPurgeStatusWaitingAgents, jobs[0].Status)
	})

	t.Run("failWhenTheMergeFails", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		config := testCdnConfig(1, "prod", []string{"www.example.com"})
		jobRepo := mockFlectoRepository.NewMockProjectCdnPurgeJobRepository(deps.ctrl)
		svc := NewCdnPurgeService(deps.appCtx, deps.configRepo, jobRepo, deps.redirectRepo, deps.pageRepo, nil)

		read := &model.ProjectCdnPurgeJob{ID: 5, CdnConfigID: 1, ProjectVersion: 1, Status: model.CdnPurgeStatusWaitingAgents}
		jobRepo.EXPECT().FindMergeable(ctx, deps.db, int64(1)).Return(read, nil)
		jobRepo.EXPECT().MergeWaiting(ctx, deps.db, read, 1).Return(false, errors.New("boom"))

		err := svc.Enqueue(ctx, deps.db, basePlan([]cdn.Entry{{Type: "BASIC", Source: "/b"}}, config), 2)

		assert.ErrorContains(t, err, "boom")
	})

	t.Run("successWithoutCap", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		deps.appCtx.Config.Cdn.MaxTargetsPerJob = 0

		plan := basePlan(
			[]cdn.Entry{{Type: "BASIC", Source: "/a"}, {Type: "BASIC", Source: "/b"}},
			testCdnConfig(1, "prod", []string{"www.example.com"}),
		)

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, plan, 1))

		var job model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.First(&job).Error)
		assert.Equal(t, model.CdnPurgeStatusWaitingAgents, job.Status)
	})

	t.Run("successSetsTheConvergenceDeadline", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		deps.appCtx.Config.Cdn.AgentConvergenceTimeout = 20 * time.Minute

		plan := basePlan([]cdn.Entry{{Type: "BASIC", Source: "/a"}}, testCdnConfig(1, "prod", []string{"www.example.com"}))
		before := time.Now()

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, plan, 1))

		var job model.ProjectCdnPurgeJob
		assert.NoError(t, deps.db.First(&job).Error)
		assert.WithinDuration(t, before.Add(20*time.Minute), job.DeadlineAt, time.Minute)
	})

	t.Run("successWithoutJobWhenNothingIsPurgeable", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()

		plan := basePlan([]cdn.Entry{{Type: "BASIC_HOST", Source: "other.com/x"}}, testCdnConfig(1, "eu", []string{"eu.example.com"}))
		plan.Entries = nil

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, plan, 1))

		var count int64
		assert.NoError(t, deps.db.Model(&model.ProjectCdnPurgeJob{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
	})

	t.Run("successWithoutConfiguration", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()

		assert.NoError(t, deps.svc.Enqueue(context.Background(), deps.db, basePlan([]cdn.Entry{{Type: "BASIC", Source: "/a"}}), 1))

		var count int64
		assert.NoError(t, deps.db.Model(&model.ProjectCdnPurgeJob{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
	})
}

func TestUnionTargets(t *testing.T) {
	tests := []struct {
		name     string
		current  []cdn.Target
		incoming []cdn.Target
		want     []cdn.Target
	}{
		{
			name:     "successWithoutIncoming",
			current:  []cdn.Target{{Kind: cdn.KindURL, Host: "a.com", Path: "/x"}},
			incoming: nil,
			want:     []cdn.Target{{Kind: cdn.KindURL, Host: "a.com", Path: "/x"}},
		},
		{
			name:     "successMergesAndSorts",
			current:  []cdn.Target{{Kind: cdn.KindURL, Host: "b.com", Path: "/x"}},
			incoming: []cdn.Target{{Kind: cdn.KindURL, Host: "a.com", Path: "/y"}},
			want: []cdn.Target{
				{Kind: cdn.KindURL, Host: "a.com", Path: "/y"},
				{Kind: cdn.KindURL, Host: "b.com", Path: "/x"},
			},
		},
		{
			name:     "successDeduplicates",
			current:  []cdn.Target{{Kind: cdn.KindURL, Host: "a.com", Path: "/x"}},
			incoming: []cdn.Target{{Kind: cdn.KindURL, Host: "a.com", Path: "/x"}},
			want:     []cdn.Target{{Kind: cdn.KindURL, Host: "a.com", Path: "/x"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, unionTargets(tt.current, tt.incoming))
		})
	}
}

func TestCdnPurgeService_PurgeURLs(t *testing.T) {
	config := testCdnConfig(1, "prod", []string{"www.example.com", "*.shop.example.com"})

	t.Run("success", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		deps.configRepo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(&config, nil)

		job, err := deps.svc.PurgeURLs(ctx, "ns1", "proj1", "prod", []string{
			"https://www.example.com/promo",
			"https://fr.shop.example.com/soldes",
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, job.Requested)
		assert.Equal(t, 0, job.ProjectVersion)
		assert.Equal(t, model.CdnPurgeStatusRunning, job.Status)
		assert.NotNil(t, job.StartedAt)
	})

	t.Run("successKeepsTheQueryStringAndPurgesBothFormsOfAnEncodedPath", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		deps.configRepo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(&config, nil)

		job, err := deps.svc.PurgeURLs(ctx, "ns1", "proj1", "prod", []string{
			"https://www.example.com/promo?utm=a",
			"https://www.example.com/caf%C3%A9",
		})

		assert.NoError(t, err)
		assert.Equal(t, []cdn.Target{
			{Kind: cdn.KindURL, Host: "www.example.com", Path: "/caf%C3%A9"},
			{Kind: cdn.KindURL, Host: "www.example.com", Path: "/café"},
			{Kind: cdn.KindURL, Host: "www.example.com", Path: "/promo?utm=a"},
		}, job.Targets)
	})

	t.Run("failWhenPurgeIsDisabledOnTheInstance", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(deps *cdnPurgeTestDeps)
		}{
			{name: "cdnDisabled", mutate: func(deps *cdnPurgeTestDeps) { deps.appCtx.Config.Cdn.Enabled = false }},
			{name: "workerDisabled", mutate: func(deps *cdnPurgeTestDeps) { deps.appCtx.Config.Cdn.WorkerInterval = 0 }},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				deps := setupCdnPurgeTest(t)
				defer deps.ctrl.Finish()
				tt.mutate(deps)

				// No configuration lookup expected.
				_, err := deps.svc.PurgeURLs(context.Background(), "ns1", "proj1", "prod", []string{"https://www.example.com/x"})

				assert.ErrorIs(t, err, ErrCdnPurgeDisabled)
			})
		}
	})

	t.Run("successDeduplicates", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		deps.configRepo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(&config, nil)

		job, err := deps.svc.PurgeURLs(ctx, "ns1", "proj1", "prod", []string{
			"https://www.example.com/promo",
			"https://www.example.com/promo",
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, job.Requested)
	})

	t.Run("successCountsUrlsItCannotSend", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		deps.configRepo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(&config, nil)

		job, err := deps.svc.PurgeURLs(ctx, "ns1", "proj1", "prod", []string{
			"https://www.example.com/promo",
			"https://other.com/x",
			"not a url",
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, job.Requested)
		assert.Equal(t, map[string]int{
			string(cdn.SkipReasonNoHostMatch): 1,
			string(cdn.SkipReasonInvalid):     1,
		}, job.Skipped)
	})

	t.Run("failWithADisabledConfiguration", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		disabled := testCdnConfig(1, "prod", []string{"www.example.com"})
		disabled.Enabled = types.Ptr(false)
		deps.configRepo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(&disabled, nil)

		_, err := deps.svc.PurgeURLs(ctx, "ns1", "proj1", "prod", []string{"https://www.example.com/x"})

		assert.ErrorIs(t, err, ErrCdnConfigDisabled)
	})

	t.Run("failWhenNothingCanBePurged", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		deps.configRepo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(&config, nil)

		_, err := deps.svc.PurgeURLs(ctx, "ns1", "proj1", "prod", []string{"https://other.com/x"})

		assert.ErrorIs(t, err, ErrCdnNothingToPurge)
		assert.ErrorContains(t, err, "no_host_match")
	})

	t.Run("failWithoutUrls", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		deps.configRepo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(&config, nil)

		_, err := deps.svc.PurgeURLs(ctx, "ns1", "proj1", "prod", nil)

		assert.ErrorContains(t, err, "no url given")
	})

	t.Run("failOverTheCap", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		deps.appCtx.Config.Cdn.MaxTargetsPerJob = 1
		deps.configRepo.EXPECT().FindByName(ctx, "ns1", "proj1", "prod").Return(&config, nil)

		_, err := deps.svc.PurgeURLs(ctx, "ns1", "proj1", "prod", []string{
			"https://www.example.com/a",
			"https://www.example.com/b",
		})

		assert.ErrorContains(t, err, "exceeds the 1 allowed")
	})

	t.Run("failWithAnUnknownConfiguration", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		deps.configRepo.EXPECT().FindByName(ctx, "ns1", "proj1", "nope").Return(nil, gorm.ErrRecordNotFound)

		_, err := deps.svc.PurgeURLs(ctx, "ns1", "proj1", "nope", []string{"https://www.example.com/x"})

		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestCdnPurgeService_History(t *testing.T) {
	t.Run("successWithTheDefaultLimit", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()

		got, err := deps.svc.History(ctx, "ns1", "proj1", 0)

		assert.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("successReturnsTheJobsNewestFirst", func(t *testing.T) {
		deps := setupCdnPurgeTest(t)
		defer deps.ctrl.Finish()
		ctx := context.Background()
		config := testCdnConfig(1, "prod", []string{"www.example.com"})

		assert.NoError(t, deps.svc.Enqueue(ctx, deps.db, model.CdnPublishPlan{
			NamespaceCode: "ns1", ProjectCode: "proj1",
			Entries: []cdn.Entry{{Type: "BASIC", Source: "/a"}},
			Configs: []model.ProjectCdnConfig{config},
		}, 1))

		got, err := deps.svc.History(ctx, "ns1", "proj1", 10)

		assert.NoError(t, err)
		assert.Len(t, got, 1)
	})
}
