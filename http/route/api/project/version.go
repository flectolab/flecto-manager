package project

import (
	"fmt"
	"net/http"

	"github.com/flectolab/flecto-manager/auth"
	"github.com/flectolab/flecto-manager/http/route"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/labstack/echo/v4"
)

func GetVersion(permissionChecker *auth.PermissionChecker, projectService service.ProjectService) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		namespaceCode := c.Param(route.NamespaceCodeKey)
		projectCode := c.Param(route.ProjectCodeKey)
		if namespaceCode == "" || projectCode == "" {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("namespaceCode and projectCode are required"))
		}
		userCtx := auth.GetUser(ctx)
		if !permissionChecker.CanResource(userCtx.SubjectPermissions, namespaceCode, projectCode, model.ResourceTypeAny, model.ActionRead) {
			return c.NoContent(http.StatusForbidden)
		}

		project, err := projectService.GetByCode(ctx, namespaceCode, projectCode)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		return c.JSON(http.StatusOK, project.Version)
	}
}
