package service

import (
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
}

func NewServices(ctx *appContext.Context, repos *repository.Repositories, jwtService *jwt.ServiceJWT) *Services {
	namespaceSrv := NewNamespaceService(ctx, repos.Namespace, repos.Project)
	projectSrv := NewProjectService(ctx, repos.Project, repos.Page, repos.RedirectDraft, repos.PageDraft)
	userSrv := NewUserService(ctx, repos.User, repos.Role)
	authSrv := NewAuthService(ctx, repos.User, jwtService)
	roleSrv := NewRoleService(ctx, repos.Role, repos.User)
	tokenSrv := NewTokenService(ctx, repos.Token, repos.Role)
	redirectSrv := NewRedirectService(ctx, repos.Redirect)
	redirectDraftSrv := NewRedirectDraftService(ctx, repos.RedirectDraft)
	redirectImportSrv := NewRedirectImportService(ctx, repos.RedirectDraft)
	pageSrv := NewPageService(ctx, repos.Page)
	pageDraftSrv := NewPageDraftService(ctx, repos.PageDraft, repos.Page)
	agentSrv := NewAgentService(ctx, repos.Agent)

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
	}
}
