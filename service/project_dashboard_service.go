package service

import (
	"context"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
)

type ProjectDashboardStats struct {
	// Project info
	Version     int
	PublishedAt *time.Time

	// Redirect stats
	RedirectTotal          int64
	RedirectCountBasic     int64
	RedirectCountBasicHost int64
	RedirectCountRegex     int64
	RedirectCountRegexHost int64

	// Redirect draft stats
	RedirectDraftTotal       int64
	RedirectDraftCountCreate int64
	RedirectDraftCountUpdate int64
	RedirectDraftCountDelete int64

	// Page stats
	PageTotal          int64
	PageCountBasic     int64
	PageCountBasicHost int64

	// Page draft stats
	PageDraftTotal       int64
	PageDraftCountCreate int64
	PageDraftCountUpdate int64
	PageDraftCountDelete int64

	// Agent stats
	AgentTotalOnline int64
	AgentCountError  int64
}

type ProjectDashboardService interface {
	GetStats(ctx context.Context, namespaceCode, projectCode string) (*ProjectDashboardStats, error)
}

type projectDashboardService struct {
	ctx                  *appContext.Context
	projectService       ProjectService
	redirectService      RedirectService
	redirectDraftService RedirectDraftService
	pageService          PageService
	pageDraftService     PageDraftService
	agentService         AgentService
}

func NewProjectDashboardService(
	ctx *appContext.Context,
	projectService ProjectService,
	redirectService RedirectService,
	redirectDraftService RedirectDraftService,
	pageService PageService,
	pageDraftService PageDraftService,
	agentService AgentService,
) ProjectDashboardService {
	return &projectDashboardService{
		ctx:                  ctx,
		projectService:       projectService,
		redirectService:      redirectService,
		redirectDraftService: redirectDraftService,
		pageService:          pageService,
		pageDraftService:     pageDraftService,
		agentService:         agentService,
	}
}

func (s *projectDashboardService) GetStats(ctx context.Context, namespaceCode, projectCode string) (*ProjectDashboardStats, error) {
	stats := &ProjectDashboardStats{}

	// Get project info
	project, err := s.projectService.GetByCode(ctx, namespaceCode, projectCode)
	if err != nil {
		return nil, err
	}
	stats.Version = project.Version
	if !project.PublishedAt.IsZero() {
		stats.PublishedAt = &project.PublishedAt
	}

	// Get redirect stats by type
	type redirectTypeCount struct {
		Type  commonTypes.RedirectType
		Count int64
	}
	var redirectCounts []redirectTypeCount
	if err = s.redirectService.GetQuery(ctx).
		Select("type, count(*) as count").
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Group("type").
		Scan(&redirectCounts).Error; err != nil {
		return nil, err
	}

	for _, rc := range redirectCounts {
		stats.RedirectTotal += rc.Count
		switch rc.Type {
		case commonTypes.RedirectTypeBasic:
			stats.RedirectCountBasic = rc.Count
		case commonTypes.RedirectTypeBasicHost:
			stats.RedirectCountBasicHost = rc.Count
		case commonTypes.RedirectTypeRegex:
			stats.RedirectCountRegex = rc.Count
		case commonTypes.RedirectTypeRegexHost:
			stats.RedirectCountRegexHost = rc.Count
		}
	}

	// Get redirect draft stats by change type
	type draftTypeCount struct {
		ChangeType model.DraftChangeType
		Count      int64
	}
	var redirectDraftCounts []draftTypeCount
	if err = s.redirectDraftService.GetQuery(ctx).
		Select("change_type, count(*) as count").
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Group("change_type").
		Scan(&redirectDraftCounts).Error; err != nil {
		return nil, err
	}

	for _, dc := range redirectDraftCounts {
		stats.RedirectDraftTotal += dc.Count
		switch dc.ChangeType {
		case model.DraftChangeTypeCreate:
			stats.RedirectDraftCountCreate = dc.Count
		case model.DraftChangeTypeUpdate:
			stats.RedirectDraftCountUpdate = dc.Count
		case model.DraftChangeTypeDelete:
			stats.RedirectDraftCountDelete = dc.Count
		}
	}

	// Get page stats by type
	type pageTypeCount struct {
		Type  commonTypes.PageType
		Count int64
	}
	var pageCounts []pageTypeCount
	if err = s.pageService.GetQuery(ctx).
		Select("type, count(*) as count").
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Group("type").
		Scan(&pageCounts).Error; err != nil {
		return nil, err
	}

	for _, pc := range pageCounts {
		stats.PageTotal += pc.Count
		switch pc.Type {
		case commonTypes.PageTypeBasic:
			stats.PageCountBasic = pc.Count
		case commonTypes.PageTypeBasicHost:
			stats.PageCountBasicHost = pc.Count
		}
	}

	// Get page draft stats by change type
	var pageDraftCounts []draftTypeCount
	if err = s.pageDraftService.GetQuery(ctx).
		Select("change_type, count(*) as count").
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Group("change_type").
		Scan(&pageDraftCounts).Error; err != nil {
		return nil, err
	}

	for _, dc := range pageDraftCounts {
		stats.PageDraftTotal += dc.Count
		switch dc.ChangeType {
		case model.DraftChangeTypeCreate:
			stats.PageDraftCountCreate = dc.Count
		case model.DraftChangeTypeUpdate:
			stats.PageDraftCountUpdate = dc.Count
		case model.DraftChangeTypeDelete:
			stats.PageDraftCountDelete = dc.Count
		}
	}

	// Get agent stats
	onlineThreshold := time.Now().Add(-s.ctx.Config.Agent.OfflineThreshold)

	// Count online agents (agents with lastHitAt > threshold)
	if err = s.agentService.GetQuery(ctx).
		Where("namespace_code = ? AND project_code = ? AND last_hit_at > ?", namespaceCode, projectCode, onlineThreshold).
		Count(&stats.AgentTotalOnline).Error; err != nil {
		return nil, err
	}

	// Count agents with error status (among online agents)
	if err = s.agentService.GetQuery(ctx).
		Where("namespace_code = ? AND project_code = ? AND status = ? AND last_hit_at > ?", namespaceCode, projectCode, commonTypes.AgentStatusError, onlineThreshold).
		Count(&stats.AgentCountError).Error; err != nil {
		return nil, err
	}

	return stats, nil
}
