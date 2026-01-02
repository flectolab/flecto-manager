package project

import (
	"fmt"
	"net/http"

	"github.com/flectolab/flecto-manager/auth"
	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/http/route"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/labstack/echo/v4"
)

func GetRedirects(permissionChecker *auth.PermissionChecker, redirectService service.RedirectService) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		namespaceCode := c.Param(route.NamespaceCodeKey)
		projectCode := c.Param(route.ProjectCodeKey)
		if namespaceCode == "" || projectCode == "" {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("namespaceCode and projectCode are required"))
		}
		userCtx := auth.GetUser(ctx)
		if !permissionChecker.CanResource(userCtx.SubjectPermissions, namespaceCode, projectCode, model.ResourceTypeRedirect, model.ActionRead) {
			return c.NoContent(http.StatusForbidden)
		}
		pagination := &commonTypes.PaginationInput{Limit: types.Ptr(500), Offset: types.Ptr(0)}
		err := c.Bind(pagination)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		redirectsDB, total, err := redirectService.FindByProjectPublished(ctx, namespaceCode, projectCode, pagination)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		redirects := make([]commonTypes.Redirect, 0)
		for _, redirect := range redirectsDB {
			redirects = append(redirects, *redirect.Redirect)
		}
		redirectList := &commonTypes.RedirectList{
			Total:  int(total),
			Offset: pagination.GetOffset(),
			Limit:  pagination.GetLimit(),
			Items:  redirects,
		}
		return c.JSON(http.StatusOK, redirectList)
	}
}
