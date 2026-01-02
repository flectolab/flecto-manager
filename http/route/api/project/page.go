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

func GetPages(permissionChecker *auth.PermissionChecker, pageService service.PageService) func(echo.Context) error {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		namespaceCode := c.Param(route.NamespaceCodeKey)
		projectCode := c.Param(route.ProjectCodeKey)
		if namespaceCode == "" || projectCode == "" {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("namespaceCode and projectCode are required"))
		}
		userCtx := auth.GetUser(ctx)
		if !permissionChecker.CanResource(userCtx.SubjectPermissions, namespaceCode, projectCode, model.ResourceTypePage, model.ActionRead) {
			return c.NoContent(http.StatusForbidden)
		}
		pagination := &commonTypes.PaginationInput{Limit: types.Ptr(500), Offset: types.Ptr(0)}
		err := c.Bind(pagination)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		pagesDB, total, err := pageService.FindByProjectPublished(ctx, namespaceCode, projectCode, pagination)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		pages := make([]commonTypes.Page, 0)
		for _, page := range pagesDB {
			pages = append(pages, *page.Page)
		}
		pageList := &commonTypes.PageList{
			Total:  int(total),
			Offset: pagination.GetOffset(),
			Limit:  pagination.GetLimit(),
			Items:  pages,
		}
		return c.JSON(http.StatusOK, pageList)
	}
}
