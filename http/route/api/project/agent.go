package project

import (
	"fmt"
	"net/http"

	"github.com/flectolab/flecto-manager/auth"
	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/http/route"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/labstack/echo/v4"
)

func PostAgent(permissionChecker *auth.PermissionChecker, agentService service.AgentService) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		namespaceCode := c.Param(route.NamespaceCodeKey)
		projectCode := c.Param(route.ProjectCodeKey)
		if namespaceCode == "" || projectCode == "" {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("namespaceCode and projectCode are required"))
		}
		userCtx := auth.GetUser(ctx)
		if !permissionChecker.CanResource(userCtx.SubjectPermissions, namespaceCode, projectCode, model.ResourceTypeAgent, model.ActionWrite) {
			return c.NoContent(http.StatusForbidden)
		}
		agentBase := commonTypes.Agent{}
		err := c.Bind(&agentBase)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		if errValidate := commonTypes.ValidateAgent(agentBase); errValidate != nil {
			return echo.NewHTTPError(http.StatusBadRequest, errValidate)
		}
		err = agentService.Upsert(ctx, &model.Agent{NamespaceCode: namespaceCode, ProjectCode: projectCode, Agent: agentBase})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		return c.NoContent(http.StatusOK)
	}
}

func PatchAgentHit(permissionChecker *auth.PermissionChecker, agentService service.AgentService) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		namespaceCode := c.Param(route.NamespaceCodeKey)
		projectCode := c.Param(route.ProjectCodeKey)
		name := c.Param(route.NameKey)
		if namespaceCode == "" || projectCode == "" || name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("namespaceCode, projectCode and name are required"))
		}
		userCtx := auth.GetUser(ctx)
		if !permissionChecker.CanResource(userCtx.SubjectPermissions, namespaceCode, projectCode, model.ResourceTypeAgent, model.ActionWrite) {
			return c.NoContent(http.StatusForbidden)
		}

		err := agentService.UpdateLastHit(ctx, namespaceCode, projectCode, name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		return c.NoContent(http.StatusOK)
	}
}
