package service

import (
	// Registers the CDN providers, looked up by type at runtime.
	_ "github.com/flectolab/flecto-manager/cdn/cloudflare"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/repository"
)

type Services struct {
	Namespace        NamespaceService
	Project          ProjectService
	User             UserService
	Auth             AuthService
	Role             RoleService
	Token            TokenService
	Redirect         RedirectService
	RedirectDraft    RedirectDraftService
	RedirectImport   RedirectImportService
	Page             PageService
	PageDraft        PageDraftService
	Agent            AgentService
	ProjectDashboard ProjectDashboardService
	Activity         ActivityService
	ProjectCdn       ProjectCdnConfigService
	CdnPurge         CdnPurgeService
	CdnPurgeWorker   CdnPurgeWorker
}

func NewServices(ctx *appContext.Context, repos *repository.Repositories, jwtService *jwt.ServiceJWT) *Services {
	activitySrv := NewActivityService(ctx, repos.ActivityEvent)
	namespaceSrv := NewNamespaceService(ctx, repos.Namespace, repos.Project)
	cdnPurgeSrv := NewCdnPurgeService(ctx, repos.ProjectCdn, repos.ProjectCdnJob, repos.Redirect, repos.Page, activitySrv)
	cdnPurgeWorker := NewCdnPurgeWorker(ctx, repos.ProjectCdnJob, repos.ProjectCdn, repos.Agent, activitySrv)
	projectSrv := NewProjectService(ctx, repos.Project, repos.Page, repos.RedirectDraft, repos.PageDraft, activitySrv, cdnPurgeSrv)
	userSrv := NewUserService(ctx, repos.User, repos.Role)
	authSrv := NewAuthService(ctx, repos.User, jwtService)
	roleSrv := NewRoleService(ctx, repos.Role, repos.User)
	tokenSrv := NewTokenService(ctx, repos.Token, repos.Role)
	redirectSrv := NewRedirectService(ctx, repos.Redirect)
	redirectDraftSrv := NewRedirectDraftService(ctx, repos.RedirectDraft, activitySrv)
	redirectImportSrv := NewRedirectImportService(ctx, repos.RedirectDraft, activitySrv)
	pageSrv := NewPageService(ctx, repos.Page)
	pageDraftSrv := NewPageDraftService(ctx, repos.PageDraft, repos.Page, activitySrv)
	agentSrv := NewAgentService(ctx, repos.Agent)
	projectCdnSrv := NewProjectCdnConfigService(ctx, repos.ProjectCdn, activitySrv)

	projectDashboardSrv := NewProjectDashboardService(
		ctx,
		projectSrv,
		redirectSrv,
		redirectDraftSrv,
		pageSrv,
		pageDraftSrv,
		agentSrv,
	)

	return &Services{
		Namespace:        namespaceSrv,
		Project:          projectSrv,
		User:             userSrv,
		Auth:             authSrv,
		Role:             roleSrv,
		Token:            tokenSrv,
		Redirect:         redirectSrv,
		RedirectDraft:    redirectDraftSrv,
		RedirectImport:   redirectImportSrv,
		Page:             pageSrv,
		PageDraft:        pageDraftSrv,
		Agent:            agentSrv,
		ProjectDashboard: projectDashboardSrv,
		Activity:         activitySrv,
		ProjectCdn:       projectCdnSrv,
		CdnPurge:         cdnPurgeSrv,
		CdnPurgeWorker:   cdnPurgeWorker,
	}
}
