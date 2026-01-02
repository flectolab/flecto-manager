package service

import (
	"context"
	"errors"
	"testing"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/config"
	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoService "github.com/flectolab/flecto-manager/mocks/flecto-manager/service"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProjectDashboardTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate tables one by one to avoid index name conflicts
	// (Agent model has a duplicate index name bug with Page model)
	_ = db.AutoMigrate(&model.Namespace{})
	_ = db.AutoMigrate(&model.Project{})
	_ = db.AutoMigrate(&model.Redirect{})
	_ = db.AutoMigrate(&model.RedirectDraft{})
	_ = db.AutoMigrate(&model.Page{})
	_ = db.AutoMigrate(&model.PageDraft{})
	// Agent table - create manually without the conflicting index
	_ = db.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		namespace_code TEXT,
		project_code TEXT,
		name TEXT,
		type TEXT,
		status TEXT,
		version INTEGER,
		error TEXT,
		load_duration INTEGER,
		last_hit_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	return db
}

func setupProjectDashboardServiceTest(t *testing.T) (
	*gomock.Controller,
	*mockFlectoService.MockProjectService,
	*mockFlectoService.MockRedirectService,
	*mockFlectoService.MockRedirectDraftService,
	*mockFlectoService.MockPageService,
	*mockFlectoService.MockPageDraftService,
	*mockFlectoService.MockAgentService,
	*gorm.DB,
	ProjectDashboardService,
) {
	ctrl := gomock.NewController(t)
	db := setupProjectDashboardTestDB(t)

	mockProjectSvc := mockFlectoService.NewMockProjectService(ctrl)
	mockRedirectSvc := mockFlectoService.NewMockRedirectService(ctrl)
	mockRedirectDraftSvc := mockFlectoService.NewMockRedirectDraftService(ctrl)
	mockPageSvc := mockFlectoService.NewMockPageService(ctrl)
	mockPageDraftSvc := mockFlectoService.NewMockPageDraftService(ctrl)
	mockAgentSvc := mockFlectoService.NewMockAgentService(ctrl)

	ctx := &appContext.Context{
		Config: &config.Config{
			Agent: config.AgentConfig{
				OfflineThreshold: 6 * time.Hour,
			},
		},
	}

	svc := NewProjectDashboardService(
		ctx,
		mockProjectSvc,
		mockRedirectSvc,
		mockRedirectDraftSvc,
		mockPageSvc,
		mockPageDraftSvc,
		mockAgentSvc,
	)

	return ctrl, mockProjectSvc, mockRedirectSvc, mockRedirectDraftSvc, mockPageSvc, mockPageDraftSvc, mockAgentSvc, db, svc
}

func TestNewProjectDashboardService(t *testing.T) {
	ctrl, _, _, _, _, _, _, _, svc := setupProjectDashboardServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
}

func TestProjectDashboardService_GetStats(t *testing.T) {
	t.Run("success with all stats", func(t *testing.T) {
		ctrl, mockProjectSvc, mockRedirectSvc, mockRedirectDraftSvc, mockPageSvc, mockPageDraftSvc, mockAgentSvc, db, svc := setupProjectDashboardServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		namespaceCode := "test-ns"
		projectCode := "test-proj"
		publishedAt := time.Now().Add(-24 * time.Hour)

		// Setup test data in DB
		// Create namespace and project first (for foreign key constraints)
		db.Create(&model.Namespace{NamespaceCode: namespaceCode, Name: "Test Namespace"})
		db.Create(&model.Project{NamespaceCode: namespaceCode, ProjectCode: projectCode, Name: "Test Project", Version: 5, PublishedAt: publishedAt})

		// Create redirects with different types
		db.Create(&model.Redirect{NamespaceCode: namespaceCode, ProjectCode: projectCode, Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/a", Target: "/b", Status: commonTypes.RedirectStatusMovedPermanent}})
		db.Create(&model.Redirect{NamespaceCode: namespaceCode, ProjectCode: projectCode, Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/c", Target: "/d", Status: commonTypes.RedirectStatusMovedPermanent}})
		db.Create(&model.Redirect{NamespaceCode: namespaceCode, ProjectCode: projectCode, Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeRegex, Source: "/e.*", Target: "/f", Status: commonTypes.RedirectStatusMovedPermanent}})
		db.Create(&model.Redirect{NamespaceCode: namespaceCode, ProjectCode: projectCode, Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasicHost, Source: "/g", Target: "/h", Status: commonTypes.RedirectStatusMovedPermanent}})
		db.Create(&model.Redirect{NamespaceCode: namespaceCode, ProjectCode: projectCode, Redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeRegexHost, Source: "/i.*", Target: "/j", Status: commonTypes.RedirectStatusMovedPermanent}})

		// Create redirect drafts with different change types
		oldRedirectID := int64(1)
		db.Create(&model.RedirectDraft{NamespaceCode: namespaceCode, ProjectCode: projectCode, ChangeType: model.DraftChangeTypeCreate, OldRedirectID: &oldRedirectID})
		db.Create(&model.RedirectDraft{NamespaceCode: namespaceCode, ProjectCode: projectCode, ChangeType: model.DraftChangeTypeCreate, OldRedirectID: &oldRedirectID})
		db.Create(&model.RedirectDraft{NamespaceCode: namespaceCode, ProjectCode: projectCode, ChangeType: model.DraftChangeTypeUpdate, OldRedirectID: &oldRedirectID})
		db.Create(&model.RedirectDraft{NamespaceCode: namespaceCode, ProjectCode: projectCode, ChangeType: model.DraftChangeTypeDelete, OldRedirectID: &oldRedirectID})

		// Create pages with different types
		db.Create(&model.Page{NamespaceCode: namespaceCode, ProjectCode: projectCode, Page: &commonTypes.Page{Type: commonTypes.PageTypeBasic, Path: "/page1", Content: "content1", ContentType: commonTypes.PageContentTypeTextPlain}})
		db.Create(&model.Page{NamespaceCode: namespaceCode, ProjectCode: projectCode, Page: &commonTypes.Page{Type: commonTypes.PageTypeBasicHost, Path: "/page2", Content: "content2", ContentType: commonTypes.PageContentTypeTextPlain}})

		// Create page drafts with different change types
		oldPageID := int64(1)
		db.Create(&model.PageDraft{NamespaceCode: namespaceCode, ProjectCode: projectCode, ChangeType: model.DraftChangeTypeCreate, OldPageID: &oldPageID})
		db.Create(&model.PageDraft{NamespaceCode: namespaceCode, ProjectCode: projectCode, ChangeType: model.DraftChangeTypeUpdate, OldPageID: &oldPageID})
		db.Create(&model.PageDraft{NamespaceCode: namespaceCode, ProjectCode: projectCode, ChangeType: model.DraftChangeTypeDelete, OldPageID: &oldPageID})

		// Create agents (online = lastHitAt within threshold)
		onlineTime := time.Now().Add(-1 * time.Hour)
		offlineTime := time.Now().Add(-12 * time.Hour)
		db.Create(&model.Agent{NamespaceCode: namespaceCode, ProjectCode: projectCode, LastHitAt: onlineTime, Agent: commonTypes.Agent{Name: "agent1", Type: commonTypes.AgentTypeTraefik, Status: commonTypes.AgentStatusSuccess}})
		db.Create(&model.Agent{NamespaceCode: namespaceCode, ProjectCode: projectCode, LastHitAt: onlineTime, Agent: commonTypes.Agent{Name: "agent2", Type: commonTypes.AgentTypeTraefik, Status: commonTypes.AgentStatusError}})
		db.Create(&model.Agent{NamespaceCode: namespaceCode, ProjectCode: projectCode, LastHitAt: offlineTime, Agent: commonTypes.Agent{Name: "agent3", Type: commonTypes.AgentTypeTraefik, Status: commonTypes.AgentStatusError}}) // offline

		// Mock expectations
		mockProjectSvc.EXPECT().
			GetByCode(ctx, namespaceCode, projectCode).
			Return(&model.Project{Version: 5, PublishedAt: publishedAt}, nil)

		mockRedirectSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Redirect{}))

		mockRedirectDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.RedirectDraft{}))

		mockPageSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Page{}))

		mockPageDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.PageDraft{}))

		mockAgentSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Agent{})).
			Times(2)

		// Execute
		stats, err := svc.GetStats(ctx, namespaceCode, projectCode)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, stats)

		// Project info
		assert.Equal(t, 5, stats.Version)
		assert.Equal(t, publishedAt.Unix(), stats.PublishedAt.Unix())

		// Redirect stats
		assert.Equal(t, int64(5), stats.RedirectTotal)
		assert.Equal(t, int64(2), stats.RedirectCountBasic)
		assert.Equal(t, int64(1), stats.RedirectCountBasicHost)
		assert.Equal(t, int64(1), stats.RedirectCountRegex)
		assert.Equal(t, int64(1), stats.RedirectCountRegexHost)

		// Redirect draft stats
		assert.Equal(t, int64(4), stats.RedirectDraftTotal)
		assert.Equal(t, int64(2), stats.RedirectDraftCountCreate)
		assert.Equal(t, int64(1), stats.RedirectDraftCountUpdate)
		assert.Equal(t, int64(1), stats.RedirectDraftCountDelete)

		// Page stats
		assert.Equal(t, int64(2), stats.PageTotal)
		assert.Equal(t, int64(1), stats.PageCountBasic)
		assert.Equal(t, int64(1), stats.PageCountBasicHost)

		// Page draft stats
		assert.Equal(t, int64(3), stats.PageDraftTotal)
		assert.Equal(t, int64(1), stats.PageDraftCountCreate)
		assert.Equal(t, int64(1), stats.PageDraftCountUpdate)
		assert.Equal(t, int64(1), stats.PageDraftCountDelete)

		// Agent stats
		assert.Equal(t, int64(2), stats.AgentTotalOnline)
		assert.Equal(t, int64(1), stats.AgentCountError)
	})

	t.Run("success with empty data", func(t *testing.T) {
		ctrl, mockProjectSvc, mockRedirectSvc, mockRedirectDraftSvc, mockPageSvc, mockPageDraftSvc, mockAgentSvc, db, svc := setupProjectDashboardServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		namespaceCode := "empty-ns"
		projectCode := "empty-proj"
		publishedAt := time.Time{}

		// Setup minimal test data
		db.Create(&model.Namespace{NamespaceCode: namespaceCode, Name: "Empty Namespace"})
		db.Create(&model.Project{NamespaceCode: namespaceCode, ProjectCode: projectCode, Name: "Empty Project", Version: 0})

		// Mock expectations
		mockProjectSvc.EXPECT().
			GetByCode(ctx, namespaceCode, projectCode).
			Return(&model.Project{Version: 0, PublishedAt: publishedAt}, nil)

		mockRedirectSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Redirect{}))

		mockRedirectDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.RedirectDraft{}))

		mockPageSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Page{}))

		mockPageDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.PageDraft{}))

		mockAgentSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Agent{})).
			Times(2)

		// Execute
		stats, err := svc.GetStats(ctx, namespaceCode, projectCode)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, stats)

		assert.Equal(t, 0, stats.Version)
		assert.Equal(t, int64(0), stats.RedirectTotal)
		assert.Equal(t, int64(0), stats.RedirectDraftTotal)
		assert.Equal(t, int64(0), stats.PageTotal)
		assert.Equal(t, int64(0), stats.PageDraftTotal)
		assert.Equal(t, int64(0), stats.AgentTotalOnline)
		assert.Equal(t, int64(0), stats.AgentCountError)
	})

	t.Run("error when project not found", func(t *testing.T) {
		ctrl, mockProjectSvc, _, _, _, _, _, _, svc := setupProjectDashboardServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		namespaceCode := "unknown-ns"
		projectCode := "unknown-proj"

		mockProjectSvc.EXPECT().
			GetByCode(ctx, namespaceCode, projectCode).
			Return(nil, errors.New("record not found"))

		// Execute
		stats, err := svc.GetStats(ctx, namespaceCode, projectCode)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "record not found")
	})

	t.Run("error when redirect query fails", func(t *testing.T) {
		ctrl, mockProjectSvc, mockRedirectSvc, _, _, _, _, db, svc := setupProjectDashboardServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		namespaceCode := "test-ns"
		projectCode := "test-proj"

		mockProjectSvc.EXPECT().
			GetByCode(ctx, namespaceCode, projectCode).
			Return(&model.Project{Version: 1}, nil)

		// Drop redirects table to cause query error
		db.Exec("DROP TABLE redirects")
		mockRedirectSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Redirect{}))

		stats, err := svc.GetStats(ctx, namespaceCode, projectCode)

		assert.Error(t, err)
		assert.Nil(t, stats)
	})

	t.Run("error when redirect draft query fails", func(t *testing.T) {
		ctrl, mockProjectSvc, mockRedirectSvc, mockRedirectDraftSvc, _, _, _, db, svc := setupProjectDashboardServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		namespaceCode := "test-ns"
		projectCode := "test-proj"

		mockProjectSvc.EXPECT().
			GetByCode(ctx, namespaceCode, projectCode).
			Return(&model.Project{Version: 1}, nil)

		mockRedirectSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Redirect{}))

		// Drop redirect_drafts table to cause query error
		db.Exec("DROP TABLE redirect_drafts")
		mockRedirectDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.RedirectDraft{}))

		stats, err := svc.GetStats(ctx, namespaceCode, projectCode)

		assert.Error(t, err)
		assert.Nil(t, stats)
	})

	t.Run("error when page query fails", func(t *testing.T) {
		ctrl, mockProjectSvc, mockRedirectSvc, mockRedirectDraftSvc, mockPageSvc, _, _, db, svc := setupProjectDashboardServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		namespaceCode := "test-ns"
		projectCode := "test-proj"

		mockProjectSvc.EXPECT().
			GetByCode(ctx, namespaceCode, projectCode).
			Return(&model.Project{Version: 1}, nil)

		mockRedirectSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Redirect{}))

		mockRedirectDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.RedirectDraft{}))

		// Drop pages table to cause query error
		db.Exec("DROP TABLE pages")
		mockPageSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Page{}))

		stats, err := svc.GetStats(ctx, namespaceCode, projectCode)

		assert.Error(t, err)
		assert.Nil(t, stats)
	})

	t.Run("error when page draft query fails", func(t *testing.T) {
		ctrl, mockProjectSvc, mockRedirectSvc, mockRedirectDraftSvc, mockPageSvc, mockPageDraftSvc, _, db, svc := setupProjectDashboardServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		namespaceCode := "test-ns"
		projectCode := "test-proj"

		mockProjectSvc.EXPECT().
			GetByCode(ctx, namespaceCode, projectCode).
			Return(&model.Project{Version: 1}, nil)

		mockRedirectSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Redirect{}))

		mockRedirectDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.RedirectDraft{}))

		mockPageSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Page{}))

		// Drop page_drafts table to cause query error
		db.Exec("DROP TABLE page_drafts")
		mockPageDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.PageDraft{}))

		stats, err := svc.GetStats(ctx, namespaceCode, projectCode)

		assert.Error(t, err)
		assert.Nil(t, stats)
	})

	t.Run("error when agent online count query fails", func(t *testing.T) {
		ctrl, mockProjectSvc, mockRedirectSvc, mockRedirectDraftSvc, mockPageSvc, mockPageDraftSvc, mockAgentSvc, db, svc := setupProjectDashboardServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		namespaceCode := "test-ns"
		projectCode := "test-proj"

		mockProjectSvc.EXPECT().
			GetByCode(ctx, namespaceCode, projectCode).
			Return(&model.Project{Version: 1}, nil)

		mockRedirectSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Redirect{}))

		mockRedirectDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.RedirectDraft{}))

		mockPageSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Page{}))

		mockPageDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.PageDraft{}))

		// Drop agents table to cause query error
		db.Exec("DROP TABLE agents")
		mockAgentSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Agent{}))

		stats, err := svc.GetStats(ctx, namespaceCode, projectCode)

		assert.Error(t, err)
		assert.Nil(t, stats)
	})

	t.Run("error when agent error count query fails", func(t *testing.T) {
		ctrl, mockProjectSvc, mockRedirectSvc, mockRedirectDraftSvc, mockPageSvc, mockPageDraftSvc, mockAgentSvc, db, svc := setupProjectDashboardServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		namespaceCode := "test-ns"
		projectCode := "test-proj"

		mockProjectSvc.EXPECT().
			GetByCode(ctx, namespaceCode, projectCode).
			Return(&model.Project{Version: 1}, nil)

		mockRedirectSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Redirect{}))

		mockRedirectDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.RedirectDraft{}))

		mockPageSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Page{}))

		mockPageDraftSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.PageDraft{}))

		// First call succeeds, then drop table for second call
		mockAgentSvc.EXPECT().
			GetQuery(ctx).
			Return(db.Model(&model.Agent{}))

		mockAgentSvc.EXPECT().
			GetQuery(ctx).
			DoAndReturn(func(ctx context.Context) *gorm.DB {
				db.Exec("DROP TABLE agents")
				return db.Model(&model.Agent{})
			})

		stats, err := svc.GetStats(ctx, namespaceCode, projectCode)

		assert.Error(t, err)
		assert.Nil(t, stats)
	})
}
