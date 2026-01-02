package resolver

import (
	"github.com/flectolab/flecto-manager/auth"
	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/graph"
	"github.com/flectolab/flecto-manager/service"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	PermissionChecker       *auth.PermissionChecker
	NamespaceService        service.NamespaceService
	ProjectService          service.ProjectService
	UserService             service.UserService
	RoleService             service.RoleService
	TokenService            service.TokenService
	RedirectService         service.RedirectService
	RedirectDraftService    service.RedirectDraftService
	RedirectImportService   service.RedirectImportService
	PageService             service.PageService
	PageDraftService        service.PageDraftService
	AgentService            service.AgentService
	ProjectDashboardService service.ProjectDashboardService
	AgentConfig             config.AgentConfig
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func convertErrorReason(reason service.ImportErrorReason) graph.ImportErrorReason {
	switch reason {
	case service.ImportErrorInvalidFormat:
		return graph.ImportErrorReasonInvalidFormat
	case service.ImportErrorInvalidType:
		return graph.ImportErrorReasonInvalidType
	case service.ImportErrorInvalidStatus:
		return graph.ImportErrorReasonInvalidStatus
	case service.ImportErrorEmptySource:
		return graph.ImportErrorReasonEmptySource
	case service.ImportErrorEmptyTarget:
		return graph.ImportErrorReasonEmptyTarget
	case service.ImportErrorDuplicateInFile:
		return graph.ImportErrorReasonDuplicateSourceInFile
	case service.ImportErrorSourceAlreadyExists:
		return graph.ImportErrorReasonSourceAlreadyExists
	case service.ImportErrorDatabaseError:
		return graph.ImportErrorReasonDatabaseError
	default:
		return graph.ImportErrorReasonInvalidFormat
	}
}
